package dbt

import (
	"fmt"
	"os"
	"path/filepath"

	v1 "github.com/getsynq/cloud/api/clients/v1"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

func ReadDbtArtifactsToReq(targetPath string) (*v1.DbtResult, error) {
	dbtResult := &v1.DbtResult{}

	manifest, invocationId, err := readArtifact(targetPath, "manifest.json")
	if err == nil {
		dbtResult.InvocationId = invocationId
		dbtResult.Manifest = wrapperspb.String(manifest)
	}

	runResults, invocationId, err := readArtifact(targetPath, "run_results.json")
	if err == nil {
		if dbtResult.InvocationId == "" {
			dbtResult.InvocationId = invocationId
		}

		dbtResult.RunResults = wrapperspb.String(runResults)
	}

	catalog, invocationId, err := readArtifact(targetPath, "catalog.json")
	if err == nil {
		if dbtResult.InvocationId == "" {
			dbtResult.InvocationId = invocationId
		}

		dbtResult.Catalog = wrapperspb.String(catalog)
	}

	sources, invocationId, err := readArtifact(targetPath, "sources.json")
	if err == nil {
		if dbtResult.InvocationId == "" {
			dbtResult.InvocationId = invocationId
		}

		dbtResult.Sources = wrapperspb.String(sources)
	}

	if manifest == "" && runResults == "" && catalog == "" && sources == "" {
		return nil, fmt.Errorf("no valid dbt artifacts found in `%s`", targetPath)
	}

	return dbtResult, nil
}

func readArtifact(directory, name string) (string, string, error) {
	artifact, err := os.ReadFile(filepath.Join(directory, name))
	if err != nil {
		logrus.Infof("synq-dbt %s, skipping", err)
		return "", "", err
	}

	invocationId := json.Get(artifact, "metadata", "invocation_id").ToString()

	logrus.Infof("synq-dbt %s found with invocation_id=`%s`", name, invocationId)

	return string(artifact), invocationId, nil
}
