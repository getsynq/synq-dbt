package command

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

var (
	exitError *exec.ExitError
)

func ExecuteCommand(cmdName string, args ...string) (exitCode int, stdOut []byte, stdErr []byte, err error) {
	cmd := exec.Command(cmdName, args...)
	stdOutReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error creating StdoutPipe for Cmd", err)
		os.Exit(1)
	}
	defer stdOutReader.Close()
	stdErrReader, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error creating StderrPipe for Cmd", err)
		os.Exit(1)
	}
	defer stdErrReader.Close()

	var outb, errb bytes.Buffer

	stdOutScanner := bufio.NewScanner(stdOutReader)
	go func() {
		for stdOutScanner.Scan() {
			t := stdOutScanner.Text()
			if len(t) == 0 {
				continue
			}
			outb.WriteString(t)
			outb.WriteByte('\n')
			fmt.Fprintln(os.Stdout, t)
		}
	}()

	stdErrScanner := bufio.NewScanner(stdErrReader)
	go func() {
		for stdErrScanner.Scan() {
			t := stdErrScanner.Text()
			if len(t) == 0 {
				continue
			}
			errb.WriteString(t)
			errb.WriteByte('\n')
			fmt.Fprintln(os.Stderr, t)
		}
	}()

	if err = cmd.Start(); err != nil {
		return 1, outb.Bytes(), errb.Bytes(), err
	}

	err = cmd.Wait()
	if err != nil {
		if errors.As(err, &exitError) {
			return exitError.ExitCode(), outb.Bytes(), errb.Bytes(), err
		}
	}

	return 0, outb.Bytes(), errb.Bytes(), nil
}
