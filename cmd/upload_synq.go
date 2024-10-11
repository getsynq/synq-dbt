package cmd

import (
	"github.com/getsynq/synq-dbt/build"
	"github.com/getsynq/synq-dbt/dbt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var SynqApiTokenFlag string
var DbtLogFile string

var uploadRunCmd = &cobra.Command{
	Use:   "synq_upload_artifacts",
	Short: "Sends to SYNQ content of dbt artifacts",
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		token := os.Getenv("SYNQ_TOKEN")
		if len(SynqApiTokenFlag) > 0 {
			token = SynqApiTokenFlag
		}
		if token == "" {
			logrus.Warnf("synq-dbt failed: missing SYNQ_TOKEN variable")
			return
		}

		targetDirectory, ok := os.LookupEnv("SYNQ_TARGET_DIR")
		if !ok {
			targetDirectory = "target"
		}

		dbtResult := dbt.CollectDbtArtifacts(targetDirectory)
		dbtResult.EnvVars = collectEnvVars()
		dbtResult.UploaderVersion = build.Version
		dbtResult.UploaderBuildTime = build.Time

		if len(DbtLogFile) > 0 {
			stdOut, _ := os.ReadFile(DbtLogFile)
			dbtResult.StdOut = stdOut
		}

		uploadArtifacts(cmd.Context(), dbtResult, token, targetDirectory)

		os.Exit(0)
	},
}

func init() {
	uploadRunCmd.Flags().StringVar(&SynqApiTokenFlag, "synq-token", "", "SYNQ API token")
	uploadRunCmd.Flags().StringVar(&DbtLogFile, "dbt-log-file", "", "File with log output of dbt command")
}
