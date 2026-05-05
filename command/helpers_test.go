package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestExecuteCommand_HangingDbt confirms the original "stuck dbt" report:
// when dbt's process hangs (e.g. Python threads blocking on Snowflake
// connection-pool cleanup), the wrapper must still return shortly after
// the context deadline. With the current implementation the chain is:
// ctx expires -> SIGINT to pgroup (sleep terminates on default action) ->
// Wait returns. If SIGINT is somehow ignored, WaitDelay escalates to
// SIGKILL automatically — covered by TestExecuteCommand_SigKillEscalation.
func TestExecuteCommand_HangingDbt(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	start := time.Now()
	code, _, _, _ := ExecuteCommand(ctx, "sleep", "60")
	elapsed := time.Since(start)

	t.Logf("ExecuteCommand returned in %v (exit code %d)", elapsed, code)

	if elapsed > 2*time.Second {
		t.Errorf(
			"ExecuteCommand took %v — context cancellation did not unblock the wrapper within 2s",
			elapsed,
		)
	}
}

// TestExecuteCommand_GrandchildNotKilled guards the fix from #16: dbt is
// placed in its own process group so a signal can hit the entire process
// tree, including grandchildren (e.g. Snowflake connector subprocesses).
func TestExecuteCommand_GrandchildNotKilled(t *testing.T) {
	ran := false
	for _, shell := range []string{"bash", "sh"} {
		if _, err := exec.LookPath(shell); err == nil {
			t.Run(shell, func(t *testing.T) {
				runGrandchildTest(t, shell)
			})
			ran = true
		}
	}
	if !ran {
		t.Skip("no shell available")
	}
}

func runGrandchildTest(t *testing.T, shell string) {
	t.Helper()

	// Shorten the SIGINT→SIGKILL escalation so the test doesn't have to
	// wait the full production grace period for unresponsive children
	// (shells set SIGINT to ignore on `cmd &` in non-interactive mode,
	// so backgrounded sleeps survive SIGINT and need the SIGKILL step).
	oldDelay := CancelGracePeriod
	CancelGracePeriod = 500 * time.Millisecond
	defer func() { CancelGracePeriod = oldDelay }()

	pidFile := t.TempDir() + "/grandchild.pid"

	// Spawn a backgrounded grandchild and a foreground sleep; record the
	// grandchild PID so the test can probe its state after cancellation.
	script := "sleep 60 & echo $! > " + pidFile + "; sleep 60"

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	ExecuteCommand(ctx, shell, "-c", script)

	time.Sleep(200 * time.Millisecond)

	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Logf("pid file not written (shell may have exited before writing): %v", err)
		return
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		t.Fatalf("invalid pid %q: %v", pidStr, err)
	}

	isAlive := syscall.Kill(pid, 0) == nil
	if isAlive {
		syscall.Kill(pid, syscall.SIGKILL)
		t.Errorf(
			"grandchild pid=%d (sleep 60) is still alive after cancellation: "+
				"signal did not reach the entire process group",
			pid,
		)
	} else {
		t.Logf("grandchild pid=%d was correctly killed", pid)
	}
}

// TestExecuteCommand_Basic verifies normal execution returns correct output and exit code.
func TestExecuteCommand_Basic(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		code, stdout, _, err := ExecuteCommand(ctx, "sh", "-c", "echo hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
		if string(stdout) != "hello\n" {
			t.Errorf("expected stdout 'hello\\n', got %q", string(stdout))
		}
	})

	t.Run("non-zero exit code", func(t *testing.T) {
		code, _, _, _ := ExecuteCommand(ctx, "sh", "-c", "exit 42")
		if code != 42 {
			t.Errorf("expected exit code 42, got %d", code)
		}
	})
}

// TestExecuteCommand_GracefulCancel: on context cancellation the wrapper
// must deliver SIGINT (not SIGKILL) so the child can run cleanup handlers.
// This is the contract dbt-snowflake relies on to cancel in-flight queries
// — sending SIGKILL strips that opportunity entirely and is the root cause
// of the "Snowflake query keeps running" customer reports.
func TestExecuteCommand_GracefulCancel(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	markerFile := t.TempDir() + "/cleanup_marker"
	// Trap SIGINT, write the marker, exit cleanly. With SIGKILL the trap
	// would never fire and the marker would be missing.
	script := fmt.Sprintf(`trap 'echo cleaned-up > %s; exit 0' INT; sleep 60`, markerFile)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, _, _, _ = ExecuteCommand(ctx, "bash", "-c", script)

	data, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf(
			"cleanup marker missing — wrapper SIGKILL'd before SIGINT handler could run: %v",
			err,
		)
	}
	if !strings.Contains(string(data), "cleaned-up") {
		t.Errorf("cleanup ran but marker contents unexpected: %q", string(data))
	}
}

// TestExecuteCommand_SigKillEscalation: when the child ignores SIGINT,
// WaitDelay must kick in and SIGKILL it so the wrapper does not hang.
// This preserves the no-hang guarantee from #16 while still giving
// well-behaved children a chance to clean up.
func TestExecuteCommand_SigKillEscalation(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	oldDelay := CancelGracePeriod
	CancelGracePeriod = 500 * time.Millisecond
	defer func() { CancelGracePeriod = oldDelay }()

	// Stubborn child: traps and ignores SIGINT, would otherwise sleep forever.
	script := "trap '' INT; sleep 60"

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, _, _, _ = ExecuteCommand(ctx, "bash", "-c", script)
	elapsed := time.Since(start)

	// Expected: ~100ms (ctx timeout) + ~500ms (WaitDelay) ≈ 600ms.
	if elapsed > 3*time.Second {
		t.Errorf("escalation to SIGKILL took too long: %v", elapsed)
	}
}

// TestExecuteCommand_OutputCaptureIntegrity: the captured stdout that ends
// up uploaded to SYNQ must match exactly what dbt printed. The previous
// bufio.Scanner implementation silently dropped blank lines, truncated on
// lines longer than 64 KiB, and lost ~30% of high-volume output to a race
// with cmd.Wait. The latter case also wedged dbt mid-run when the kernel
// pipe buffer filled and writes blocked.
func TestExecuteCommand_OutputCaptureIntegrity(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	t.Run("blank lines preserved", func(t *testing.T) {
		ctx := context.Background()
		_, stdout, _, _ := ExecuteCommand(ctx, "sh", "-c", "printf 'a\\n\\nb\\n'")
		if string(stdout) != "a\n\nb\n" {
			t.Errorf("blank line dropped: got %q want %q", string(stdout), "a\n\nb\n")
		}
	})

	t.Run("long line preserved", func(t *testing.T) {
		ctx := context.Background()
		// 100 KiB single line, then a marker on the next line. The previous
		// scanner gave up at 64 KiB and silently swallowed the rest.
		script := "head -c 102400 /dev/zero | tr '\\0' 'a'; echo; echo MARKER"
		_, stdout, _, _ := ExecuteCommand(ctx, "sh", "-c", script)
		if !strings.Contains(string(stdout), "MARKER") {
			t.Errorf(
				"long line truncated capture silently — got %d bytes, MARKER missing",
				len(stdout),
			)
		}
	})

	t.Run("no truncation under load", func(t *testing.T) {
		// Repro for the cmd.Wait/scanner-goroutine race: the previous
		// implementation lost 25-40% of lines on every run.
		const lines = 5000
		for i := 0; i < 5; i++ {
			ctx := context.Background()
			script := fmt.Sprintf("for i in $(seq 1 %d); do echo line$i; done", lines)
			_, stdout, _, _ := ExecuteCommand(ctx, "sh", "-c", script)
			got := strings.Count(string(stdout), "\n")
			if got != lines {
				t.Errorf("iter %d: expected %d lines, got %d", i, lines, got)
			}
		}
	})
}

// TestExecuteCommand_PythonSnowflakeHarness exercises the full
// dbt-snowflake-style cancellation contract: a Python child registers a
// SIGINT handler that "cancels" a long-running query in the same way
// snowflake.connector.cursor.cancel() does, the wrapper is asked to
// cancel mid-flight, and we assert the handler ran AND the query state
// flipped to "cancelled" instead of running to completion.
//
// This is the closest we can get to the customer's bug without a real
// Snowflake account; an end-to-end test against `SYSTEM$WAIT` is the
// natural follow-up.
func TestExecuteCommand_PythonSnowflakeHarness(t *testing.T) {
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available")
	}

	script, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	script = script + "/testdata/snowflake_cancel_probe.py"
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("probe script missing at %s: %v", script, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type result struct {
		stdout string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		_, stdout, _, err := ExecuteCommand(ctx, py, script)
		done <- result{string(stdout), err}
	}()

	// Wait long enough for the script to register its handler and start
	// the simulated long query.
	time.Sleep(500 * time.Millisecond)
	cancel()

	var r result
	select {
	case r = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("script did not exit within 10s — signal not delivered or process not killed")
	}

	if !strings.Contains(r.stdout, "STARTED") {
		t.Fatalf("script never started:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "SIGNAL_RECEIVED=") {
		t.Errorf(
			"script never saw a signal — wrapper SIGKILL'd before the handler could run:\n%s",
			r.stdout,
		)
	}
	if !strings.Contains(r.stdout, "FINAL_STATE=cancelled") {
		t.Errorf(
			"cursor.cancel() did not run — a real Snowflake query would leak:\n%s",
			r.stdout,
		)
	}
	if strings.Contains(r.stdout, "COMPLETED") {
		t.Errorf("script ran to completion despite cancellation:\n%s", r.stdout)
	}
}
