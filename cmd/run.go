package cmd

import (
	"errors"
	"os"

	"github.com/getsynq/synq-dbt/command"
	"github.com/getsynq/synq-dbt/dbt"
	"github.com/getsynq/synq-dbt/synq"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use: "dbt",
	RunE: func(cmd *cobra.Command, args []string) error {
		url := "dbt-uploader-xwpzuoapgq-lm.a.run.app:443"
		targetDirectory := "target"
		dbtBin := "dbt"

		exitCode, err := command.ExecuteCommand(dbtBin, args...)

		token, ok := os.LookupEnv("SYNQ_TOKEN")
		if !ok {
			return errors.New("environment variable SYNQ_TOKEN was not set")
		}

		if _, err := os.Stat(targetDirectory); !os.IsNotExist(err) {
			return err
		}

		client, err := synq.CreateDbtServiceClient(url)
		if err != nil {
			return err
		}
		api := synq.Api{
			DbtClient: client,
		}

		dbtArtifacts, err := dbt.ReadDbtArtifactsToReq(targetDirectory)
		if err != nil {
			return err
		}

		dbtArtifacts.Token = token

		err = api.SendRequest(cmd.Context(), dbtArtifacts)
		if err != nil {
			return err
		}

		os.Exit(exitCode)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
