package cmd

import (
	"context"
	v1 "github.com/getsynq/cloud/api/clients/v1"
	"github.com/getsynq/synq-dbt/build"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"os"
	"strings"

	"github.com/getsynq/synq-dbt/command"
	"github.com/getsynq/synq-dbt/dbt"
	"github.com/getsynq/synq-dbt/synq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	SilenceUsage:       true,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		token, ok := os.LookupEnv("SYNQ_TOKEN")
		if !ok {
			logrus.Printf("synq-dbt failed: missing SYNQ_TOKEN variable")
		}

		url, ok := os.LookupEnv("SYNQ_UPLOAD_URL")
		if !ok {
			url = "dbt-uploader-xwpzuoapgq-lm.a.run.app:443"
		}

		targetDirectory, ok := os.LookupEnv("SYNQ_TARGET_DIR")
		if !ok {
			targetDirectory = "target"
		}

		dbtBin, ok := os.LookupEnv("SYNQ_DBT_BIN")
		if !ok {
			dbtBin = "dbt"
		}

		// Collect all arguments including flags
		args = os.Args[1:]

		logrus.Infof("synq-dbt processing `%s %s`", dbtBin, strings.Join(args, " "))

		exitCode, stdOut, stdErr, err := command.ExecuteCommand(dbtBin, args...)
		if err != nil {
			logrus.Printf("synq-dbt execution of dbt finished with exit code %d, %s", exitCode, err.Error())
		}

		if token != "" {
			logrus.Infof("synq-dbt processing `%s`, uploading to `%s`", targetDirectory, url)

			dbtResult := dbt.CollectDbtArtifacts(targetDirectory)
			dbtResult.StdOut = stdOut
			dbtResult.StdErr = stdErr
			dbtResult.EnvVars = collectEnvVars()
			dbtResult.UploaderVersion = build.Version
			dbtResult.UploaderBuildTime = build.Time
			dbtResult.Args = args
			dbtResult.ExitCode = wrapperspb.Int32(int32(exitCode))

			if err := uploadArtifactsToSynq(cmd.Context(), dbtResult, token, url); err != nil {
				logrus.Printf("synq-dbt failed: %s", err.Error())
			}
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

func uploadArtifactsToSynq(ctx context.Context, dbtResult *v1.DbtResult, token, url string) error {
	api, err := synq.NewApi(url)
	if err != nil {
		return err
	}

	dbtResult.Token = token

	err = api.SendRequest(ctx, dbtResult)
	if err != nil {
		return err
	}

	logrus.Infof("synq-dbt successful")

	return nil
}
