package synq

import (
	"context"

	ingestdbtv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/ingest/dbt/v1"
	"github.com/getsynq/synq-dbt/dbt"
	"github.com/getsynq/synq-dbt/git"
)

// RequestBuilder helps construct an IngestInvocationRequest.
type RequestBuilder struct {
	request *ingestdbtv1.IngestInvocationRequest
}

// NewRequestBuilder creates a new RequestBuilder.
func NewRequestBuilder() *RequestBuilder {
	return &RequestBuilder{
		request: &ingestdbtv1.IngestInvocationRequest{},
	}
}

// WithArtifacts adds dbt artifacts to the request.
func (b *RequestBuilder) WithArtifacts(artifacts *dbt.Artifacts) *RequestBuilder {
	if artifacts == nil {
		return b
	}

	var dbtArtifacts []*ingestdbtv1.DbtArtifact
	if len(artifacts.Manifest) > 0 {
		dbtArtifacts = append(dbtArtifacts, &ingestdbtv1.DbtArtifact{
			Artifact: &ingestdbtv1.DbtArtifact_ManifestJson{
				ManifestJson: []byte(artifacts.Manifest),
			},
		})
	}
	if len(artifacts.RunResults) > 0 {
		dbtArtifacts = append(dbtArtifacts, &ingestdbtv1.DbtArtifact{
			Artifact: &ingestdbtv1.DbtArtifact_RunResultsJson{
				RunResultsJson: []byte(artifacts.RunResults),
			},
		})
	}
	if len(artifacts.Sources) > 0 {
		dbtArtifacts = append(dbtArtifacts, &ingestdbtv1.DbtArtifact{
			Artifact: &ingestdbtv1.DbtArtifact_SourcesJson{
				SourcesJson: []byte(artifacts.Sources),
			},
		})
	}
	if len(artifacts.Catalog) > 0 {
		dbtArtifacts = append(dbtArtifacts, &ingestdbtv1.DbtArtifact{
			Artifact: &ingestdbtv1.DbtArtifact_CatalogJson{
				CatalogJson: []byte(artifacts.Catalog),
			},
		})
	}
	b.request.Artifacts = dbtArtifacts
	return b
}

// WithEnvVars adds environment variables to the request.
func (b *RequestBuilder) WithEnvVars(envVars map[string]string) *RequestBuilder {
	b.request.EnvironmentVars = envVars
	return b
}

// WithUploaderInfo adds uploader version and build time.
func (b *RequestBuilder) WithUploaderInfo(version, buildTime string) *RequestBuilder {
	b.request.UploaderVersion = version
	b.request.UploaderBuildTime = buildTime
	return b
}

// WithStdOut adds stdout to the request.
func (b *RequestBuilder) WithStdOut(stdOut []byte) *RequestBuilder {
	b.request.StdOut = stdOut
	return b
}

// WithStdErr adds stderr to the request.
func (b *RequestBuilder) WithStdErr(stdErr []byte) *RequestBuilder {
	b.request.StdErr = stdErr
	return b
}

// WithArgs adds command arguments to the request.
func (b *RequestBuilder) WithArgs(args []string) *RequestBuilder {
	b.request.Args = args
	return b
}

// WithExitCode adds the exit code to the request.
func (b *RequestBuilder) WithExitCode(exitCode int) *RequestBuilder {
	b.request.ExitCode = int32(exitCode)
	return b
}

// WithGitContext collects and adds git context from the specified directory.
func (b *RequestBuilder) WithGitContext(ctx context.Context, dir string) *RequestBuilder {
	b.request.GitContext = git.CollectGitContext(ctx, dir)
	return b
}

// Build returns the constructed IngestInvocationRequest.
func (b *RequestBuilder) Build() *ingestdbtv1.IngestInvocationRequest {
	return b.request
}
