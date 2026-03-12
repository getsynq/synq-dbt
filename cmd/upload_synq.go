package cmd

import (
	"os"

	"github.com/getsynq/synq-dbt/build"
	"github.com/getsynq/synq-dbt/dbt"
	"github.com/getsynq/synq-dbt/synq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
			logrus.Errorf("synq-dbt failed: missing SYNQ_TOKEN variable")
			return
		}

		targetDirectory := dbt.ResolveTargetDir(nil)
		artifacts := dbt.CollectDbtArtifacts(targetDirectory)

		builder := synq.NewRequestBuilder().
			WithArtifacts(artifacts).
			WithEnvVars(collectEnvVars()).
			WithUploaderInfo(build.Version, build.Time).
			WithGitContext(cmd.Context(), ".")

		if len(DbtLogFile) > 0 {
			stdOut, _ := os.ReadFile(DbtLogFile)
			builder.WithStdOut(stdOut)
		}

		synq.UploadArtifacts(cmd.Context(), builder.Build(), token, targetDirectory)

		os.Exit(0)
	},
}

func init() {
	uploadRunCmd.Flags().StringVar(&SynqApiTokenFlag, "synq-token", "", "SYNQ API token")
	uploadRunCmd.Flags().StringVar(&DbtLogFile, "dbt-log-file", "", "File with log output of dbt command")
}
