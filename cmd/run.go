package cmd

import (
	ingestdbtv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/ingest/dbt/v1"
	"context"
	"github.com/getsynq/synq-dbt/git"
	"os"
	"strconv"
	"strings"

	v1 "github.com/getsynq/cloud/api/clients/v1"
	"github.com/getsynq/synq-dbt/build"
	"google.golang.org/protobuf/types/known/wrapperspb"

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
		if !ok || token == "" {
			logrus.Warnf("synq-dbt failed: missing SYNQ_TOKEN variable")
		}

		targetDirectory, ok := os.LookupEnv("SYNQ_TARGET_DIR")
		if !ok {
			targetDirectory = "target"
		}

		dbtBin, ok := os.LookupEnv("SYNQ_DBT_BIN")
		dbtBin = strings.TrimSpace(dbtBin)
		if !ok || dbtBin == "" {
			dbtBin = "dbt"
		}

		// Collect all arguments including flags
		args = os.Args[1:]

		logrus.Infof("synq-dbt processing `%s`", strings.Join(append([]string{dbtBin}, args...), " "))

		exitCode, stdOut, stdErr, err := command.ExecuteCommand(cmd.Context(), dbtBin, args...)
		if err != nil {
			logrus.Warnf("synq-dbt execution of dbt finished with exit code %d, %s", exitCode, err.Error())
		}

		if token != "" {
			dbtResult := dbt.CollectDbtArtifacts(targetDirectory)
			dbtResult.StdOut = stdOut
			dbtResult.StdErr = stdErr
			dbtResult.EnvVars = collectEnvVars()
			dbtResult.UploaderVersion = build.Version
			dbtResult.UploaderBuildTime = build.Time
			dbtResult.Args = args
			dbtResult.ExitCode = wrapperspb.Int32(int32(exitCode))

			uploadArtifacts(cmd.Context(), dbtResult, token, targetDirectory)
		}

		os.Exit(exitCode)
	},
}

var EnvsToCollect = map[string]struct{}{
	"AIRFLOW_CTX_DAG_OWNER":      {},
	"AIRFLOW_CTX_DAG_ID":         {},
	"AIRFLOW_CTX_TASK_ID":        {},
	"AIRFLOW_CTX_EXECUTION_DATE": {},
	"AIRFLOW_CTX_TRY_NUMBER":     {},
	"AIRFLOW_CTX_DAG_RUN_ID":     {},
}

func uploadArtifacts(ctx context.Context, dbtResult *v1.DbtResult, token string, targetDirectory string) {
	synqV1ApiEndpoint, ok := os.LookupEnv("SYNQ_UPLOAD_URL")
	if !ok {
		synqV1ApiEndpoint = "dbtapi.synq.io:443"
	}
	synqV2ApiEndpoint := "https://developer.synq.io/"
	if envEndpoint, ok := os.LookupEnv("SYNQ_API_ENDPOINT"); ok {
		synqV2ApiEndpoint = envEndpoint
	}

	var err error
	useSYNQApiV2, _ := strconv.ParseBool(os.Getenv("SYNQ_API_V2"))
	useSYNQApiV2 = useSYNQApiV2 || strings.HasPrefix(token, "st-")
	if useSYNQApiV2 {
		logrus.Infof("synq-dbt processing `%s`, uploading to `%s` using v2 API", targetDirectory, synqV2ApiEndpoint)
		err = uploadArtifactsToSYNQV2(ctx, dbtResult, token, synqV2ApiEndpoint)
	} else {
		logrus.Infof("synq-dbt processing `%s`, uploading to `%s`", targetDirectory, synqV1ApiEndpoint)
		err = uploadArtifactsToSYNQ(ctx, dbtResult, token, synqV1ApiEndpoint)
	}

	if err != nil {
		logrus.Warnf("synq-dbt failed: %s", err.Error())
	} else {
		logrus.Info("synq-dbt processing successfully finished")
	}
}

func collectEnvVars() map[string]string {
	envs := map[string]string{}
	for envName := range EnvsToCollect {
		envValue := os.Getenv(envName)
		if len(envValue) > 0 {
			envs[envName] = envValue
		}
	}
	return envs
}

func uploadArtifactsToSYNQ(ctx context.Context, dbtResult *v1.DbtResult, token, url string) error {
	api, err := synq.NewApi(url)
	if err != nil {
		return err
	}

	dbtResult.Token = token

	err = api.SendRequest(ctx, dbtResult)
	if err != nil {
		return err
	}

	logrus.Infof("synq-dbt successful")

	return nil
}

func uploadArtifactsToSYNQV2(ctx context.Context, dbtResult *v1.DbtResult, token string, synqApiEndpoint string) error {
	if dbtResult == nil || token == "" {
		return nil
	}

	var artifacts []*ingestdbtv1.DbtArtifact
	if dbtResult.Manifest != nil && len(dbtResult.Manifest.Value) > 0 {
		artifacts = append(artifacts, &ingestdbtv1.DbtArtifact{
			Artifact: &ingestdbtv1.DbtArtifact_ManifestJson{
				ManifestJson: []byte(dbtResult.Manifest.GetValue()),
			},
		})
	}
	if dbtResult.RunResults != nil && len(dbtResult.RunResults.Value) > 0 {
		artifacts = append(artifacts, &ingestdbtv1.DbtArtifact{
			Artifact: &ingestdbtv1.DbtArtifact_RunResultsJson{
				RunResultsJson: []byte(dbtResult.RunResults.GetValue()),
			},
		})
	}
	if dbtResult.Sources != nil && len(dbtResult.Sources.Value) > 0 {
		artifacts = append(artifacts, &ingestdbtv1.DbtArtifact{
			Artifact: &ingestdbtv1.DbtArtifact_SourcesJson{
				SourcesJson: []byte(dbtResult.Sources.GetValue()),
			},
		})
	}
	if dbtResult.Catalog != nil && len(dbtResult.Catalog.Value) > 0 {
		artifacts = append(artifacts, &ingestdbtv1.DbtArtifact{
			Artifact: &ingestdbtv1.DbtArtifact_CatalogJson{
				CatalogJson: []byte(dbtResult.Catalog.GetValue()),
			},
		})
	}

	dbtInvocation := &ingestdbtv1.IngestInvocationRequest{
		Args:              dbtResult.GetArgs(),
		ExitCode:          dbtResult.GetExitCode().GetValue(),
		StdOut:            dbtResult.GetStdOut(),
		StdErr:            dbtResult.GetStdErr(),
		EnvironmentVars:   dbtResult.GetEnvVars(),
		Artifacts:         artifacts,
		UploaderVersion:   dbtResult.GetUploaderVersion(),
		UploaderBuildTime: dbtResult.GetUploaderBuildTime(),
		GitContext:        git.CollectGitContext(ctx, "."),
	}

	return synq.UploadMetadata(ctx, dbtInvocation, synqApiEndpoint, token)
}
