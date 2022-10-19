package command

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	dbtv1 "github.com/getsynq/cloud/api/clients/dbt/v1"
	v1 "github.com/getsynq/cloud/api/clients/v1"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
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

type DbtApi interface {
	PostDbtResult(ctx context.Context, in *dbtv1.PostDbtResultRequest, opts ...grpc.CallOption) (*dbtv1.PostDbtResultResponse, error)
}

type Api struct {
	DbtClient DbtApi
}

func CreateDbtClient() (dbtv1.DbtServiceClient, error) {
	systemRoots, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	creds := credentials.NewTLS(&tls.Config{
		RootCAs: systemRoots,
	})

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	var uploadUrl string
	if !wantsStaging {
		uploadUrl = prodUrl
	} else {
		uploadUrl = stagingUrl
	}

	conn, err := grpc.Dial(uploadUrl, opts...)
	if err != nil {
		return nil, err
	}

	return dbtv1.NewDbtServiceClient(conn), nil
}

func ProcessTargetCmd(cmd *cobra.Command, args []string) error {
	tokenVal, ok := os.LookupEnv("SYNQ_TOKEN")
	if !ok {
		return errors.New("environment variable SYNQ_TOKEN was not set")
	}
	synqToken = tokenVal
	if _, err := os.Stat(targetDirectory); !os.IsNotExist(err) {
		log("Directory found. Start processing..", logrus.InfoLevel)

		client, err := CreateDbtClient()
		if err != nil {
			return err
		}
		api := Api{
			DbtClient: client,
		}

		err = api.ProcessResults(synqToken, targetDirectory)
		if err != nil {
			return err
		}
	} else {
		log(fmt.Sprintf("Directory '%s' containing run results couldn't be found", targetDirectory), logrus.ErrorLevel)
	}
	if dbtCmdExitCode != 0 {
		log("Passing around dbt exit code", logrus.InfoLevel)
		os.Exit(dbtCmdExitCode)
	}
	return nil
}

func (api *Api) ProcessResults(token, targetPath string) error {
	manifestReader, err := fileToReader(filepath.Join(targetPath, "manifest.json"))
	if err != nil {
		return err
	}
	runResultsReader, err := fileToReader(filepath.Join(targetPath, "run_results.json"))
	if err != nil {
		return err
	}
	catalogReader, err := fileToReader(filepath.Join(targetPath, "catalog.json"))
	if err != nil {
		return err
	}
	sourcesReader, err := fileToReader(filepath.Join(targetPath, "sources.json"))
	if err != nil {
		return err
	}

	err = api.sendRequest(context.Background(), token, manifestReader, runResultsReader, catalogReader, sourcesReader)
	if err != nil {
		//_, copyErr := io.Copy(os.Stdout, resp.Body)
		//if copyErr != nil {
		//	return copyErr
		//}
		return err
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

func (api *Api) sendRequest(ctx context.Context, token string, manifest, runResults, catalog, sources *io.PipeReader) error {
	dbtResult := &v1.DbtResult{
		Token: token,
	}

	var manifestInvocationId string

	if manifest != nil {
		bytes, err := GetBytes(manifest)
		if err != nil {
			return fmt.Errorf("problem parsing manifest.json: %+v", err)
		}
		manifestInvocationId = json.Get(bytes, "metadata", "invocation_id").ToString()
		dbtResult.InvocationId = manifestInvocationId
		dbtResult.Manifest = wrapperspb.String(string(bytes))
	}

	if runResults != nil {
		bytes, err := GetBytes(runResults)
		if err != nil {
			return fmt.Errorf("problem parsing run_results.json: %+v", err)
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
			return fmt.Errorf("problem parsing catalog.json: %+v", err)
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
			return fmt.Errorf("problem parsing catalog.json: %+v", err)
		}
		sourcesInvocationId := json.Get(bytes, "metadata", "invocation_id").ToString()
		if dbtResult.InvocationId == "" {
			dbtResult.InvocationId = sourcesInvocationId
		}
		dbtResult.Sources = wrapperspb.String(string(bytes))
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if dbtResult.Manifest != nil || dbtResult.RunResults != nil || dbtResult.Catalog != nil || dbtResult.Sources != nil {
		_, err := api.DbtClient.PostDbtResult(timeoutCtx, &dbtv1.PostDbtResultRequest{
			DbtResult: dbtResult,
		})
		if err != nil {
			return err
		}
		log("All done!", logrus.InfoLevel)
	} else {
		log("Nothing to upload", logrus.ErrorLevel)
	}

	return nil
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
