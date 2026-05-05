package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/getsynq/synq-dbt/build"
	"github.com/getsynq/synq-dbt/command"
	"github.com/getsynq/synq-dbt/dbt"
	"github.com/getsynq/synq-dbt/synq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// uploadArtifactsSafe runs the SYNQ-side upload pipeline with a panic guard
// so that any failure on our side (artifact parsing, gRPC, OAuth, …) is
// swallowed and never affects dbt's exit code propagation. The wrapper is
// supposed to be transparent: dbt has already finished by the time we get
// here, and the orchestrator must see dbt's real exit code.
func uploadArtifactsSafe(
	ctx context.Context,
	token string,
	args []string,
	exitCode int,
	stdOut, stdErr []byte,
) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("synq-dbt: panic during upload (ignored): %v", r)
		}
	}()

	targetDirectory := dbt.ResolveTargetDir(args)
	artifacts := dbt.CollectDbtArtifacts(targetDirectory)

	request := synq.NewRequestBuilder().
		WithArtifacts(artifacts).
		WithStdOut(stdOut).
		WithStdErr(stdErr).
		WithEnvVars(collectEnvVars()).
		WithUploaderInfo(build.Version, build.Time).
		WithArgs(args).
		WithExitCode(exitCode).
		WithGitContext(ctx, ".").
		Build()

	synq.UploadArtifacts(ctx, request, token, targetDirectory)
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	SilenceUsage:       true,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		token, ok := os.LookupEnv("SYNQ_TOKEN")
		if !ok || token == "" {
			logrus.Warnf("synq-dbt failed: missing SYNQ_TOKEN variable")
		}

		dbtBin, ok := os.LookupEnv("SYNQ_DBT_BIN")
		dbtBin = strings.TrimSpace(dbtBin)
		if !ok || dbtBin == "" {
			dbtBin = "dbt"
		}

		// Collect all arguments including flags
		args = os.Args[1:]

		logrus.Infof("synq-dbt processing `%s`", strings.Join(append([]string{dbtBin}, args...), " "))

		exitCode, stdOut, stdErr, err := command.ExecuteCommand(cmd.Context(), dbtBin, args...)
		if err != nil {
			logrus.Warnf("synq-dbt execution of dbt finished with exit code %d, %s", exitCode, err.Error())
		}

		if token != "" {
			uploadArtifactsSafe(cmd.Context(), token, args, exitCode, stdOut, stdErr)
		}

		os.Exit(exitCode)
	},
}

var EnvsToCollect = map[string]struct{}{
	"AIRFLOW_CTX_DAG_OWNER":      {},
	"AIRFLOW_CTX_DAG_ID":         {},
	"AIRFLOW_CTX_TASK_ID":        {},
	"AIRFLOW_CTX_EXECUTION_DATE": {},
	"AIRFLOW_CTX_TRY_NUMBER":     {},
	"AIRFLOW_CTX_DAG_RUN_ID":     {},
}

func collectEnvVars() map[string]string {
	envs := map[string]string{}
	for envName := range EnvsToCollect {
		envValue := os.Getenv(envName)
		if len(envValue) > 0 {
			envs[envName] = envValue
		}
	}
	return envs
}
