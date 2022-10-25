package cmd

import (
	"context"
	"errors"
	"os"
	"strings"

	"log"

	"github.com/getsynq/synq-dbt/command"
	"github.com/getsynq/synq-dbt/dbt"
	"github.com/getsynq/synq-dbt/synq"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	SilenceUsage:       true,
	RunE: func(cmd *cobra.Command, args []string) error {

		// Collect all arguments including flags
		args = os.Args[1:]

		log.Printf("syn-dbt processing `dbt %s`", strings.Join(args, " "))

		dbtBin := "dbt"

		exitCode, err := command.ExecuteCommand(dbtBin, args...)

		if err != nil {
			os.Exit(exitCode)
			return nil
		}

		if err := uploadArtifactsToSynq(cmd.Context()); err != nil {
			log.Printf("syn-dbt failed: %s", err.Error())
		}

		log.Printf("syn-dbt successful")

		return nil
	},
}

func init() {}

func uploadArtifactsToSynq(ctx context.Context) error {
	targetDirectory := "target"
	url := "dbt-uploader-xwpzuoapgq-lm.a.run.app:443"

	log.Printf("syn-dbt processing `%s`, uploading to `%s`", targetDirectory, url)

	token, ok := os.LookupEnv("SYNQ_TOKEN")
	if !ok {
		return errors.New("environment variable SYNQ_TOKEN was not set")
	}

	if _, err := os.Stat(targetDirectory); !os.IsNotExist(err) {
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

	return api.SendRequest(ctx, dbtArtifactsReq)
}
