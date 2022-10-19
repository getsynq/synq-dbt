package command

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
)

var (
	exitError *exec.ExitError
	logger    = logrus.WithField("app", "synq-client")
)

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func WrapCommand(cmdName string, args ...string) (error, int) {
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
			fmt.Println(stdOutScanner.Text())
		}
	}()

	stdErrScanner := bufio.NewScanner(stdErrReader)
	go func() {
		for stdErrScanner.Scan() {
			fmt.Println(stdErrScanner.Text())
		}
	}()

	if err = cmd.Start(); err != nil {
		return err, 1
	}

	err = cmd.Wait()
	if err != nil {
		if errors.As(err, &exitError) {
			return err, exitError.ExitCode()
		}
	}

	return nil, 0
}

func log(msg string, level logrus.Level) {
	logger.Log(level, msg)
}
