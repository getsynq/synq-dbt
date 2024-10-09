package main

import (
	"context"
	"github.com/getsynq/synq-dbt/build"
	"github.com/getsynq/synq-dbt/cmd"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

//go:generate bash bin/version.sh

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	signals := []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signals...)
	go func() {
		select {
		case sig := <-ch:
			logrus.Printf("synq-dbt received %s signal, shutting down", sig.String())
			cancel()
		case <-ctx.Done():
		}
	}()
	defer cancel()

	logrus.SetFormatter(&easy.Formatter{
		TimestampFormat: "15:04:05",
		LogFormat:       "%time%  %msg%\n",
	})

	logrus.Printf("synq-dbt %s (%s) started (pid %d pgrp %d ppid %d)", strings.TrimSpace(build.Version), strings.TrimSpace(build.Time), os.Getpid(), syscall.Getpgrp(), syscall.Getppid())

	cmd.Execute(ctx)
}
