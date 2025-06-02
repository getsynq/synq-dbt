package synq

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"

	ingestdbtv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/ingest/dbt/v1/dbtv1grpc"
	ingestdbtv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/ingest/dbt/v1"
	v1 "github.com/getsynq/cloud/api/clients/v1"
	"github.com/getsynq/synq-dbt/git"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func uploadArtifactsToSYNQV2(ctx context.Context, dbtResult *v1.DbtResult, token string, synqApiEndpoint string) error {
	if dbtResult == nil || token == "" {
		return nil
	}

	dbtInvocation := dbtResultToIngestInvocationRequest(ctx, dbtResult)

	err := ingestInvocation(ctx, dbtInvocation, synqApiEndpoint, token)
	if err != nil {
		return err
	}

	logrus.Infof("synq-dbt upload successful (v2)")

	return nil
}

func ingestInvocation(ctx context.Context, output *ingestdbtv1.IngestInvocationRequest, endpoint string, token string) error {
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	oauthTokenSource, err := LongLivedTokenSource(ctx, token, parsedEndpoint)
	if err != nil {
		return err
	}
	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: false})
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(oauthTokenSource),
		grpc.WithAuthority(parsedEndpoint.Host),
	}

	conn, err := grpc.DialContext(ctx, grpcEndpoint(parsedEndpoint), opts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	dbtServiceClient := ingestdbtv1grpc.NewDbtServiceClient(conn)

	resp, err := dbtServiceClient.IngestInvocation(ctx, output)
	if err != nil {
		return err
	}
	logrus.Printf("metadata uploaded successfully: %s", resp.String())
	return nil
}

func grpcEndpoint(endpoint *url.URL) string {
	port := endpoint.Port()
	if port == "" {
		port = "443"
	}
	return fmt.Sprintf("%s:%s", endpoint.Hostname(), port)
}

func dbtResultToIngestInvocationRequest(ctx context.Context, dbtResult *v1.DbtResult) *ingestdbtv1.IngestInvocationRequest {
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
	return dbtInvocation
}
