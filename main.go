package main

import "github.com/getsynq/cloud/synq-clients/commanders/dbt/command"

//go:generate ../../../bin/dev-tools protos --protos ../../../proto

func main() {
	command.Execute()
}
