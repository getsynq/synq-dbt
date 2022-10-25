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

	manifest, err := os.ReadFile(filepath.Join(targetPath, "manifest.json"))
	if err == nil {
		manifestInvocationId := json.Get(manifest, "metadata", "invocation_id").ToString()
		logrus.Infof("syn-dbt manifest.json found with invocation_id=`%s`", manifestInvocationId)

		dbtResult.InvocationId = manifestInvocationId
		dbtResult.Manifest = wrapperspb.String(string(manifest))
	} else {
		logrus.Infof("syn-dbt %s", err)
	}

	runResults, err := os.ReadFile(filepath.Join(targetPath, "run_results.json"))
	if err == nil {
		runResultsInvocationId := json.Get(runResults, "metadata", "invocation_id").ToString()
		logrus.Infof("syn-dbt run_results.json found with invocation_id=`%s`", runResultsInvocationId)

		if dbtResult.InvocationId == "" {
			dbtResult.InvocationId = runResultsInvocationId
		}
		dbtResult.RunResults = wrapperspb.String(string(runResults))
	} else {
		logrus.Infof("syn-dbt %s", err)
	}

	catalog, err := os.ReadFile(filepath.Join(targetPath, "catalog.json"))
	if err == nil {
		catalogInvocationId := json.Get(catalog, "metadata", "invocation_id").ToString()
		logrus.Infof("syn-dbt catalog.json found with invocation_id=`%s`", catalogInvocationId)

		if dbtResult.InvocationId == "" {
			dbtResult.InvocationId = catalogInvocationId
		}
		dbtResult.Catalog = wrapperspb.String(string(catalog))
	} else {
		logrus.Infof("syn-dbt %s", err)
	}

	sources, err := os.ReadFile(filepath.Join(targetPath, "sources.json"))
	if err == nil {
		sourcesInvocationId := json.Get(sources, "metadata", "invocation_id").ToString()
		logrus.Infof("syn-dbt sources.json found with invocation_id=`%s`", sourcesInvocationId)

		if dbtResult.InvocationId == "" {
			dbtResult.InvocationId = sourcesInvocationId
		}
		dbtResult.Sources = wrapperspb.String(string(sources))
	} else {
		logrus.Infof("syn-dbt %s", err)
	}

	if manifest == nil && runResults == nil && catalog == nil && sources == nil {
		return nil, fmt.Errorf("no valid dbt artifacts found in `%s`", targetPath)
	}

	return dbtResult, nil
}
