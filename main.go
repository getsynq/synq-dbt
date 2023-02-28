package main

import (
	"github.com/getsynq/synq-dbt/build"
	"strings"

	"github.com/getsynq/synq-dbt/cmd"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

//go:generate bash bin/version.sh

func main() {
	logrus.SetFormatter(&easy.Formatter{
		TimestampFormat: "15:04:05",
		LogFormat:       "%time%  %msg%\n",
	})

	logrus.Printf("synq-dbt %s (%s) started", strings.TrimSpace(build.Version), strings.TrimSpace(build.Time))

	cmd.Execute()
}
