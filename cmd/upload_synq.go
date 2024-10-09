package cmd

import (
	"github.com/getsynq/synq-dbt/build"
	"github.com/getsynq/synq-dbt/dbt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var uploadRunCmd = &cobra.Command{
	Use:   "upload_artifacts",
	Short: "Sends to SYNQ content of dbt artifacts",
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		token, ok := os.LookupEnv("SYNQ_TOKEN")
		if !ok || token == "" {
			logrus.Warnf("synq-dbt failed: missing SYNQ_TOKEN variable")
			return
		}

		targetDirectory, ok := os.LookupEnv("SYNQ_TARGET_DIR")
		if !ok {
			targetDirectory = "target"
		}

		if token != "" {
			dbtResult := dbt.CollectDbtArtifacts(targetDirectory)
			dbtResult.EnvVars = collectEnvVars()
			dbtResult.UploaderVersion = build.Version
			dbtResult.UploaderBuildTime = build.Time
			uploadArtifacts(cmd.Context(), dbtResult, token, targetDirectory)
		}

		os.Exit(0)
	},
}

func init() {
	runCmd.AddCommand(uploadRunCmd)
}
