package cmd

import (
	"context"
	"errors"
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

		// Collect all arguments including flags
		args = os.Args[1:]

		logrus.Infof("syn-dbt processing `dbt %s`", strings.Join(args, " "))

		dbtBin := "dbt"

		exitCode, err := command.ExecuteCommand(dbtBin, args...)
		if err != nil {
			os.Exit(exitCode)
		}

		if err := uploadArtifactsToSynq(cmd.Context()); err != nil {
			logrus.Printf("syn-dbt failed: %s", err.Error())
		}

		os.Exit(exitCode)
	},
}

func uploadArtifactsToSynq(ctx context.Context) error {
	targetDirectory := "target"
	url := "dbt-uploader-xwpzuoapgq-lm.a.run.app:443"

	logrus.Infof("syn-dbt processing `%s`, uploading to `%s`", targetDirectory, url)

	token, ok := os.LookupEnv("SYNQ_TOKEN")
	if !ok {
		return errors.New("environment variable SYNQ_TOKEN was not set")
	}

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

	logrus.Infof("syn-dbt successful")

	return nil
}
