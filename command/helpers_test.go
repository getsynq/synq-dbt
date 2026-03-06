package command

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestExecuteCommand_HangingDbt confirms the root cause of a reported hang:
// when dbt's process hangs (e.g. Python threads blocking on Snowflake connection
// pool cleanup), cmd.Wait() blocks indefinitely.
//
// The fix is to give the context a timeout so that once the deadline passes,
// cmd.Cancel kills the stuck dbt process and ExecuteCommand returns promptly.
// Without a context timeout on the dbt command, synq-dbt waits forever.
func TestExecuteCommand_HangingDbt(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not available")
	}

	// Context with a short timeout simulates Airflow's execution_timeout
	// eventually signalling the task.
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	start := time.Now()
	// "sleep 60" simulates a dbt process stuck in Python cleanup threads.
	code, _, _, _ := ExecuteCommand(ctx, "sleep", "60")
	elapsed := time.Since(start)

	t.Logf("ExecuteCommand returned in %v (exit code %d)", elapsed, code)

	// Must return shortly after the 300ms deadline, not 60 seconds later.
	if elapsed > 2*time.Second {
		t.Errorf("ExecuteCommand took %v — context cancellation did not kill the stuck process within 2s", elapsed)
	}
}

// TestExecuteCommand_GrandchildNotKilled demonstrates the process-group bug:
// Pgid=Getpgrp() puts dbt in synq-dbt's process group. When cmd.Cancel
// calls syscall.Kill(-cmd.Process.Pid, SIGKILL), it targets a process group
// with ID = dbt's PID. But dbt's actual PGID is synq-dbt's PGID (not dbt's
// PID), so the kill hits the wrong group and grandchildren survive.
//
// With the fix (Pgid=0), dbt gets its own process group (PGID = dbt's PID),
// and Cancel's group kill correctly terminates all grandchildren.
func TestExecuteCommand_GrandchildNotKilled(t *testing.T) {
	for _, shell := range []string{"bash", "sh"} {
		if _, err := exec.LookPath(shell); err == nil {
			t.Run(shell, func(t *testing.T) {
				runGrandchildTest(t, shell)
			})
			continue
		}
	}
	t.Skip("no shell available")
}

func runGrandchildTest(t *testing.T, shell string) {
	t.Helper()
	pidFile := t.TempDir() + "/grandchild.pid"

	// Script behavior:
	//  1. Spawns a grandchild (sleep 60) that inherits the process group
	//  2. Writes the grandchild PID to pidFile
	//  3. The parent shell ALSO sleeps (simulating dbt stuck in Python cleanup)
	// This ensures the context timeout fires while the parent is still alive,
	// triggering cmd.Cancel. We then check whether the grandchild was killed.
	script := "sleep 60 & echo $! > " + pidFile + "; sleep 60"

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	ExecuteCommand(ctx, shell, "-c", script)

	// Give the grandchild a moment to be written; it may still be starting.
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

	// Signal 0 checks existence without sending a real signal.
	isAlive := syscall.Kill(pid, 0) == nil
	if isAlive {
		// Clean up the grandchild to avoid leaving it running.
		syscall.Kill(pid, syscall.SIGKILL)

		t.Errorf(
			"grandchild pid=%d (sleep 60) is still alive after context cancellation: "+
				"cmd.Cancel's syscall.Kill(-childPID, SIGKILL) targets the wrong PGID "+
				"because Pgid=Getpgrp() puts dbt in synq-dbt's process group, not its own",
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
