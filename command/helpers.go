package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// CancelGracePeriod is how long dbt has to react to SIGINT (run its
// KeyboardInterrupt handler — which is what cancels in-flight Snowflake /
// database queries and closes connections) before we escalate to SIGKILL on
// the entire process group.
//
// Kept comfortably below typical orchestrator kill timeouts so the wrapper
// finishes its own cleanup before the orchestrator gives up on us:
//
//   - Airflow's killed_task_cleanup_time defaults to 60s
//   - Kubernetes default terminationGracePeriodSeconds is 30s
//
// Snowflake's ABORT QUERY and dbt-core's KeyboardInterrupt cleanup both
// complete in a few seconds in practice, so 15s leaves plenty of head-room.
// Exposed as a var so tests can shorten it.
var CancelGracePeriod = 15 * time.Second

func ExecuteCommand(
	ctx context.Context,
	cmdName string,
	args ...string,
) (exitCode int, stdOut []byte, stdErr []byte, err error) {
	cmd := exec.CommandContext(ctx, cmdName, args...)

	// Pgid: 0 puts dbt in its own process group (PGID = dbt's PID) so we
	// can deliver signals to the entire process tree, including grandchildren
	// such as the Snowflake connector subprocesses.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	// killTimer is shared between cmd.Cancel (which arms it) and the post-
	// Wait cleanup (which stops it if the process exited cleanly).
	var killTimer *time.Timer

	// On context cancellation we want a graceful-then-forceful escalation:
	//
	//  1. SIGINT to the whole process group lets dbt run its
	//     KeyboardInterrupt handler — which is what calls
	//     snowflake.connector.cursor.cancel() to abort in-flight Snowflake
	//     queries. Sending SIGKILL straight away (the previous behaviour)
	//     skipped this entirely and is the cause of the "Snowflake query
	//     keeps running after a cancelled run" customer reports.
	//
	//  2. After CancelGracePeriod, SIGKILL the whole process group. This
	//     mops up grandchildren that ignore SIGINT — including
	//     shell-backgrounded jobs (bash sets SIGINT to ignore on `cmd &`
	//     in non-interactive mode) and any third-party subprocess that
	//     intentionally traps and discards SIGINT. This preserves the
	//     no-hang guarantee from #16: an unresponsive process tree is
	//     guaranteed to be gone after CancelGracePeriod.
	cmd.Cancel = func() error {
		pgid := -cmd.Process.Pid
		logrus.Printf(
			"cancelling subcommand pgid=%d (SIGINT, escalating to SIGKILL after %s)",
			cmd.Process.Pid,
			CancelGracePeriod,
		)
		killTimer = time.AfterFunc(CancelGracePeriod, func() {
			logrus.Printf("subcommand pgid=%d still alive after grace period — SIGKILL", -pgid)
			_ = syscall.Kill(pgid, syscall.SIGKILL)
		})
		if err := syscall.Kill(pgid, syscall.SIGINT); err != nil {
			// ESRCH means the group is already gone — not an error.
			if errors.Is(err, syscall.ESRCH) {
				return nil
			}
			return fmt.Errorf("SIGINT to pgid=%d: %w", -pgid, err)
		}
		return nil
	}
	// WaitDelay is the upper bound for how long Wait() will block on I/O
	// after the process has exited (or after Cancel has fired). Set it
	// slightly longer than the grace period so our SIGKILL has a chance to
	// take effect before Go force-closes the pipes.
	cmd.WaitDelay = CancelGracePeriod + 2*time.Second

	// Capture stdout/stderr while still mirroring them to the user's
	// terminal. io.MultiWriter avoids the bufio.Scanner pitfalls of the
	// previous implementation: race with cmd.Wait, silent truncation on
	// lines >64 KiB, dropped blank lines, and — most importantly — the
	// deadlock where a too-long line stops the scanner, the kernel pipe
	// buffer fills, and dbt blocks mid-write unable to do any cleanup.
	// cmd.Wait synchronises with the internal copy goroutines, so reading
	// the buffers afterwards is race-free.
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	if err := cmd.Start(); err != nil {
		return -1, nil, nil, fmt.Errorf("starting %s: %w", cmdName, err)
	}
	logrus.Printf("subcommand pid=%d", cmd.Process.Pid)

	waitErr := cmd.Wait()

	// Stop the SIGKILL timer if it hasn't fired yet — the process exited
	// before the grace period elapsed.
	if killTimer != nil {
		killTimer.Stop()
	}

	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return exitErr.ExitCode(), stdoutBuf.Bytes(), stderrBuf.Bytes(), waitErr
		}
		// Non-ExitError (ErrWaitDelay, broken pipe, etc.) — surface the
		// error rather than silently returning exit-code 0 success.
		return -1, stdoutBuf.Bytes(), stderrBuf.Bytes(), waitErr
	}

	return 0, stdoutBuf.Bytes(), stderrBuf.Bytes(), nil
}
