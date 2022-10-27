package main

import (
	_ "embed"
	"strings"

	"github.com/getsynq/synq-dbt/cmd"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

//go:generate bash bin/version.sh

//go:embed version.txt
var version string

//go:embed build.txt
var build string

func main() {
	logrus.SetFormatter(&easy.Formatter{
		TimestampFormat: "15:04:05",
		LogFormat:       "%time%  %msg%\n",
	})

	logrus.Printf("synq-dbt %s (%s) started", strings.TrimSpace(version), strings.TrimSpace(build))

	cmd.Execute()
}
