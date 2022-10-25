package command

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/getsynq/cloud/synq-clients/commanders/dbt/dbt"
	"github.com/getsynq/cloud/synq-clients/commanders/dbt/synq"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	//"time"
)

func noWriteJson(cmd *cobra.Command, args []string) error {
	if contains(args, "--no-write-json") {
		return errors.New("--no-write-json flag cannot be used with Synq.\nWe process the run results to send them to your Synq workspace")
	} else {
		return nil
	}
}

var (
	json            = jsoniter.ConfigCompatibleWithStandardLibrary
	prodUrl         string
	stagingUrl      string
	wantsStaging    bool
	dbtBin          string
	dbtCmdExitCode  int
	synqToken       string
	targetDirectory string
	DbtExecCmd      = &cobra.Command{
		Short: "Use synq dbt command to execute any DBT command.",
		Long: "Synq dbt cli tool is a wrapper around dbt executable that parses your dbt run / test results and publish them to your Synq workspace.\n" +
			"Use this command as you would normally execute your dbt executable - only remove the `dbt` prefix and pass all the arguments after double-dash `--`.\n" +
			"For Example: `dbt run --target=production --models finance` becomes `synq dbt --token=yourSynqToken -- run --target=production --models=finance`",
		Example: "SYNQ_TOKEN=xyz dbt run --models finance",
		Use:     "dbt",
		Aliases: []string{},
		Args:    noWriteJson,
		Run: func(cmd *cobra.Command, args []string) {
			err, exitCode := WrapCommand(dbtBin, args...)
			dbtCmdExitCode = exitCode
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		},
		PostRunE: ProcessTargetCmd,
	}
	dbtUploadCmd = &cobra.Command{
		Short: "Uploads dbt results from the target directory to synq",
		Long: "Synq upload command uploads resulting manifest files from dbt run that reside in --target-dir (by default `target`). When choosing synq upload command \n" +
			"make sure to run upload no matter the result of dbt. In case your pipeline executes dbt, that fails and synq upload is not called. The UI will show only successfull runs",
		Example: "synq upload --token=xyz",
		Use:     "upload",
		Aliases: []string{},
		Args:    noWriteJson,
		RunE:    ProcessTargetCmd,
	}
)

func ProcessTargetCmd(cmd *cobra.Command, args []string) error {
	token, ok := os.LookupEnv("SYNQ_TOKEN")
	if !ok {
		return errors.New("environment variable SYNQ_TOKEN was not set")
	}

	if _, err := os.Stat(targetDirectory); !os.IsNotExist(err) {
		log("Directory found. Start processing..", logrus.InfoLevel)

		client, err := synq.CreateDbtServiceClient(prodUrl)
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

		return api.SendRequest(context.Background(), dbtArtifacts)

	} else {
		log(fmt.Sprintf("Directory '%s' containing run results couldn't be found", targetDirectory), logrus.ErrorLevel)
	}
	if dbtCmdExitCode != 0 {
		log("Passing around dbt exit code", logrus.InfoLevel)
		os.Exit(dbtCmdExitCode)
	}
	return nil
}

func Execute() {
	DbtExecCmd.Execute()
}

func DbtCommand() *cobra.Command {
	return DbtExecCmd
}

func DbtUploadCommand() *cobra.Command {
	return dbtUploadCmd
}

func init() {
	DbtExecCmd.Flags().StringVar(&synqToken, "token", "", "Your Synq connection token")
	DbtExecCmd.Flags().StringVar(&dbtBin, "dbt-bin", "dbt", "Location of dbt binary")
	DbtExecCmd.Flags().StringVar(&prodUrl, "prod-url", "dbt-uploader-xwpzuoapgq-lm.a.run.app:443", "Upload url for the dbt run results")
	DbtExecCmd.Flags().StringVar(&stagingUrl, "staging-url", "dbt-uploader-hcdlgjmqkq-lm.a.run.app:443", "Upload url for the dbt run results")
	DbtExecCmd.Flags().BoolVar(&wantsStaging, "staging", false, "This uploads the data to synq staging")
	DbtExecCmd.Flags().StringVar(&targetDirectory, "target-dir", "target", "Target directory, where DBT stores manifest.json")

	err := DbtExecCmd.Flags().MarkHidden("prod-url")
	if err != nil {
		return
	}
	err = DbtExecCmd.Flags().MarkHidden("staging-url")
	if err != nil {
		return
	}
	err = DbtExecCmd.Flags().MarkHidden("staging")
	if err != nil {
		return
	}
}
