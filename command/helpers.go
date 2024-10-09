package command

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

var (
	exitError *exec.ExitError
)

func ExecuteCommand(ctx context.Context, cmdName string, args ...string) (exitCode int, stdOut []byte, stdErr []byte, err error) {
	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    syscall.Getpgrp(),
	}
	cmd.Cancel = func() error {
		logrus.Println("cancelling subcommand", cmd.Process.Pid)
		var errors []error
		if err := cmd.Process.Kill(); err != nil {
			errors = append(errors, err)
		}
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			errors = append(errors, err)
		}
		if len(errors) > 0 {
			return fmt.Errorf("error cancelling pid=%d, errors=%v", cmd.Process.Pid, errors)
		}
		return nil
	}
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
	if cmd.Process != nil {
		logrus.Printf("subcommand pid=%d", cmd.Process.Pid)
	}

	err = cmd.Wait()
	if err != nil {
		if errors.As(err, &exitError) {
			return exitError.ExitCode(), outb.Bytes(), errb.Bytes(), err
		}
	}

	return 0, outb.Bytes(), errb.Bytes(), nil
}
