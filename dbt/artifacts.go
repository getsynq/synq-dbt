package dbt

import (
	stdjson "encoding/json"
	"fmt"
	"os"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

func CollectDbtArtifacts(targetPath string) *Artifacts {
	artifacts := &Artifacts{}

	manifest, invocationId, err := readArtifact(targetPath, "manifest.json")
	if err == nil {
		artifacts.InvocationId = invocationId
		artifacts.Manifest = manifest
	}

	runResults, invocationId, err := readArtifact(targetPath, "run_results.json")
	if err == nil {
		if artifacts.InvocationId == "" {
			artifacts.InvocationId = invocationId
		}

		artifacts.RunResults = runResults
	}

	catalog, invocationId, err := readArtifact(targetPath, "catalog.json")
	if err == nil {
		if artifacts.InvocationId == "" {
			artifacts.InvocationId = invocationId
		}

		artifacts.Catalog = catalog
	}

	sources, invocationId, err := readArtifact(targetPath, "sources.json")
	if err == nil {
		if artifacts.InvocationId == "" {
			artifacts.InvocationId = invocationId
		}

		artifacts.Sources = sources
	}

	return artifacts
}

func readArtifact(directory, name string) (string, string, error) {
	artifact, err := os.ReadFile(filepath.Join(directory, name))
	if err != nil {
		logrus.Infof("synq-dbt %s, skipping", err)
		return "", "", err
	}

	if len(artifact) == 0 {
		logrus.Warnf("synq-dbt %s is empty, skipping", name)
		return "", "", fmt.Errorf("%s is empty", name)
	}

	if !stdjson.Valid(artifact) {
		logrus.Warnf("synq-dbt %s contains invalid JSON (file may have been modified during read), skipping", name)
		return "", "", fmt.Errorf("%s contains invalid JSON", name)
	}

	invocationId := json.Get(artifact, "metadata", "invocation_id").ToString()

	logrus.Infof("synq-dbt %s found with invocation_id=`%s`", name, invocationId)

	return string(artifact), invocationId, nil
}
