package dbt

import (
	"fmt"
	"os"
	"path/filepath"

	v1 "github.com/getsynq/cloud/api/clients/v1"
	jsoniter "github.com/json-iterator/go"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

func ReadDbtArtifactsToReq(targetPath string) (*v1.DbtResult, error) {
	dbtResult := &v1.DbtResult{}

	manifest, err := os.ReadFile(filepath.Join(targetPath, "manifest.json"))
	if err == nil {
		manifestInvocationId := json.Get(manifest, "metadata", "invocation_id").ToString()
		dbtResult.InvocationId = manifestInvocationId
		dbtResult.Manifest = wrapperspb.String(string(manifest))
	}

	runResults, err := os.ReadFile(filepath.Join(targetPath, "run_results.json"))
	if err == nil {
		runResultsInvocationId := json.Get(runResults, "metadata", "invocation_id").ToString()
		if dbtResult.InvocationId == "" {
			dbtResult.InvocationId = runResultsInvocationId
		}
		dbtResult.RunResults = wrapperspb.String(string(runResults))
	}

	catalog, err := os.ReadFile(filepath.Join(targetPath, "catalog.json"))
	if err == nil {
		catalogInvocationId := json.Get(catalog, "metadata", "invocation_id").ToString()
		if dbtResult.InvocationId == "" {
			dbtResult.InvocationId = catalogInvocationId
		}
		dbtResult.Catalog = wrapperspb.String(string(catalog))
	}

	sources, err := os.ReadFile(filepath.Join(targetPath, "sources.json"))
	if err != nil {
		sourcesInvocationId := json.Get(sources, "metadata", "invocation_id").ToString()
		if dbtResult.InvocationId == "" {
			dbtResult.InvocationId = sourcesInvocationId
		}
		dbtResult.Sources = wrapperspb.String(string(sources))
	}

	if manifest != nil || runResults != nil || catalog != nil || sources != nil {
		return nil, fmt.Errorf("no valid dbt artifacts found in `%s`", targetPath)
	}

	return dbtResult, nil
}
