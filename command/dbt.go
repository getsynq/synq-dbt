package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	v1 "github.com/getsynq/cloud/api/clients/v1"
	"github.com/getsynq/cloud/synq-clients/commanders/dbt/synq"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/wrapperspb"
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

		dbtArtifacts, err := buildDbtArtifactsReq(context.Background(), targetDirectory)
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

func fileToReader(filePath string) (*io.PipeReader, error) {
	r, w := io.Pipe()
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		log("file not found "+filePath, logrus.WarnLevel)
		return nil, nil
	} else {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}

		go func() {
			if _, err := io.Copy(w, file); err != nil {
				fmt.Println(err.Error())
			}
			defer file.Close()
			defer w.Close()
		}()
		return r, nil
	}
}

func GetBytes(r *io.PipeReader) ([]byte, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if !json.Valid(b) {
		return nil, fmt.Errorf("is not a valid json")
	}

	return b, nil
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

func buildDbtArtifactsReq(ctx context.Context, targetPath string) (*v1.DbtResult, error) {
	manifest, err := fileToReader(filepath.Join(targetPath, "manifest.json"))
	if err != nil {
		return nil, err
	}
	runResults, err := fileToReader(filepath.Join(targetPath, "run_results.json"))
	if err != nil {
		return nil, err
	}
	catalog, err := fileToReader(filepath.Join(targetPath, "catalog.json"))
	if err != nil {
		return nil, err
	}
	sources, err := fileToReader(filepath.Join(targetPath, "sources.json"))
	if err != nil {
		return nil, err
	}

	dbtResult := &v1.DbtResult{}

	var manifestInvocationId string

	if manifest != nil {
		bytes, err := GetBytes(manifest)
		if err != nil {
			return nil, fmt.Errorf("problem parsing manifest.json: %+v", err)
		}
		manifestInvocationId = json.Get(bytes, "metadata", "invocation_id").ToString()
		dbtResult.InvocationId = manifestInvocationId
		dbtResult.Manifest = wrapperspb.String(string(bytes))
	}

	if runResults != nil {
		bytes, err := GetBytes(runResults)
		if err != nil {
			return nil, fmt.Errorf("problem parsing run_results.json: %+v", err)
		}
		runResultsInvocationId := json.Get(bytes, "metadata", "invocation_id").ToString()
		if dbtResult.InvocationId == "" {
			dbtResult.InvocationId = runResultsInvocationId
		}
		dbtResult.RunResults = wrapperspb.String(string(bytes))
	}

	if catalog != nil {
		bytes, err := GetBytes(catalog)
		if err != nil {
			return nil, fmt.Errorf("problem parsing catalog.json: %+v", err)
		}
		catalogInvocationId := json.Get(bytes, "metadata", "invocation_id").ToString()
		if dbtResult.InvocationId == "" {
			dbtResult.InvocationId = catalogInvocationId
		}
		dbtResult.Catalog = wrapperspb.String(string(bytes))
	}

	if sources != nil {
		bytes, err := GetBytes(sources)
		if err != nil {
			return nil, fmt.Errorf("problem parsing catalog.json: %+v", err)
		}
		sourcesInvocationId := json.Get(bytes, "metadata", "invocation_id").ToString()
		if dbtResult.InvocationId == "" {
			dbtResult.InvocationId = sourcesInvocationId
		}
		dbtResult.Sources = wrapperspb.String(string(bytes))
	}

	return dbtResult, nil

}

// 	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
// 	defer cancel()

// 	if dbtResult.Manifest != nil || dbtResult.RunResults != nil || dbtResult.Catalog != nil || dbtResult.Sources != nil {
// 		_, err := api.DbtClient.PostDbtResult(timeoutCtx, &dbtv1.PostDbtResultRequest{
// 			DbtResult: dbtResult,
// 		})
// 		if err != nil {
// 			return err
// 		}
// 		log("All done!", logrus.InfoLevel)
// 	} else {
// 		log("Nothing to upload", logrus.ErrorLevel)
// 	}

// 	return nil
// }
