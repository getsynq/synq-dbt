package cmd

import (
	"context"
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

		exitCode, err := command.ExecuteCommand(dbtBin, args...)
		if err != nil {
			logrus.Printf("synq-dbt execution of dbt finished with exit code %d, %s", exitCode, err.Error())
		}

		if token != "" {
			if err := uploadArtifactsToSynq(cmd.Context(), targetDirectory, token, url); err != nil {
				logrus.Printf("synq-dbt failed: %s", err.Error())
			}
		}

		os.Exit(exitCode)
	},
}

func uploadArtifactsToSynq(ctx context.Context, targetDirectory, token, url string) error {
	logrus.Infof("synq-dbt processing `%s`, uploading to `%s`", targetDirectory, url)

	if _, err := os.Stat(targetDirectory); os.IsNotExist(err) {
		return err
	}

	api, err := synq.NewApi(url)
	if err != nil {
		return err
	}

	dbtArtifactsReq, err := dbt.ReadDbtArtifactsToReq(targetDirectory)
	if err != nil {
		return err
	}

	dbtArtifactsReq.Token = token

	err = api.SendRequest(ctx, dbtArtifactsReq)
	if err != nil {
		return err
	}

	logrus.Infof("synq-dbt successful")

	return nil
}
