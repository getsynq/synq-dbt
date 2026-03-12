package dbt

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// ResolveTargetDir determines the dbt target directory by checking multiple sources
// in priority order:
//  1. SYNQ_TARGET_DIR env var (explicit SYNQ override)
//  2. --target-path from dbt CLI args (only in wrap mode)
//  3. DBT_TARGET_PATH env var (dbt's own env var)
//  4. target-path in dbt_project.yml (legacy dbt config)
//  5. "target" (dbt default)
func ResolveTargetDir(dbtArgs []string) string {
	if dir, ok := os.LookupEnv("SYNQ_TARGET_DIR"); ok {
		logrus.Infof("synq-dbt using target directory from SYNQ_TARGET_DIR: %s", dir)
		return dir
	}

	if dir := parseTargetPathFromArgs(dbtArgs); dir != "" {
		logrus.Infof("synq-dbt using target directory from --target-path flag: %s", dir)
		return dir
	}

	if dir, ok := os.LookupEnv("DBT_TARGET_PATH"); ok {
		logrus.Infof("synq-dbt using target directory from DBT_TARGET_PATH: %s", dir)
		return dir
	}

	if dir := readTargetPathFromProject("dbt_project.yml"); dir != "" {
		logrus.Infof("synq-dbt using target directory from dbt_project.yml: %s", dir)
		return dir
	}

	return "target"
}

// parseTargetPathFromArgs extracts the --target-path value from dbt CLI arguments.
func parseTargetPathFromArgs(args []string) string {
	for i, arg := range args {
		if arg == "--target-path" && i+1 < len(args) {
			return args[i+1]
		}
		// Handle --target-path=value form
		if strings.HasPrefix(arg, "--target-path=") {
			return strings.TrimPrefix(arg, "--target-path=")
		}
	}
	return ""
}

// dbtProjectConfig represents the relevant fields from dbt_project.yml.
type dbtProjectConfig struct {
	TargetPath string `yaml:"target-path"`
}

// readTargetPathFromProject reads target-path from a dbt_project.yml file.
// Returns empty string if the file doesn't exist or can't be parsed.
func readTargetPathFromProject(projectPath string) string {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}

	var config dbtProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		logrus.Debugf("synq-dbt failed to parse %s: %v", projectPath, err)
		return ""
	}

	return config.TargetPath
}
