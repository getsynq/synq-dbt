package main

import (
	"github.com/getsynq/synq-dbt/cmd"
)

//go:generate ../../../bin/dev-tools protos --protos ../../../proto

func main() {
	cmd.Execute()
}
