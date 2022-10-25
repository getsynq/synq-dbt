package command

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

var (
	exitError *exec.ExitError
)

func ExecuteCommand(cmdName string, args ...string) (int, error) {
	cmd := exec.Command(cmdName, args...)
	stdOutReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error creating StdoutPipe for Cmd", err)
		os.Exit(1)
	}
	stdErrReader, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error creating StderrPipe for Cmd", err)
		os.Exit(1)
	}

	stdOutScanner := bufio.NewScanner(stdOutReader)
	go func() {
		for stdOutScanner.Scan() {
			fmt.Fprintln(os.Stdout, stdOutScanner.Text())
		}
	}()

	stdErrScanner := bufio.NewScanner(stdErrReader)
	go func() {
		for stdErrScanner.Scan() {
			fmt.Fprintln(os.Stderr, stdErrScanner.Text())
		}
	}()

	if err = cmd.Start(); err != nil {
		return 1, err
	}

	err = cmd.Wait()
	if err != nil {
		if errors.As(err, &exitError) {
			return exitError.ExitCode(), err
		}
	}

	return 0, nil
}
