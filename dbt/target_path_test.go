package dbt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTargetPathFromArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no args", nil, ""},
		{"no target-path", []string{"run", "--profiles-dir", "/tmp"}, ""},
		{"separate value", []string{"run", "--target-path", "/custom/target"}, "/custom/target"},
		{"equals form", []string{"run", "--target-path=/custom/target"}, "/custom/target"},
		{"at end without value", []string{"run", "--target-path"}, ""},
		{"with other flags around", []string{"run", "--select", "model", "--target-path", "out", "--full-refresh"}, "out"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTargetPathFromArgs(tt.args)
			if got != tt.want {
				t.Errorf("parseTargetPathFromArgs(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestReadTargetPathFromProject(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"with target-path", "name: my_project\ntarget-path: custom_target\n", "custom_target"},
		{"with quoted target-path", "name: my_project\ntarget-path: \"custom_target\"\n", "custom_target"},
		{"without target-path", "name: my_project\nversion: '1.0'\n", ""},
		{"empty target-path", "name: my_project\ntarget-path: \n", ""},
		{"invalid yaml", ":::invalid\n", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "dbt_project.yml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			got := readTargetPathFromProject(path)
			if got != tt.want {
				t.Errorf("readTargetPathFromProject() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadTargetPathFromProject_MissingFile(t *testing.T) {
	got := readTargetPathFromProject("/nonexistent/dbt_project.yml")
	if got != "" {
		t.Errorf("expected empty string for missing file, got %q", got)
	}
}

func TestResolveTargetDir_Priority(t *testing.T) {
	// Setup a dbt_project.yml with target-path
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "dbt_project.yml")
	os.WriteFile(projectPath, []byte("target-path: from_yaml\n"), 0644)

	// Save and restore working directory
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Clean env
	os.Unsetenv("SYNQ_TARGET_DIR")
	os.Unsetenv("DBT_TARGET_PATH")

	// Test: default falls through to yaml
	got := ResolveTargetDir(nil)
	if got != "from_yaml" {
		t.Errorf("expected 'from_yaml', got %q", got)
	}

	// Test: DBT_TARGET_PATH takes precedence over yaml
	os.Setenv("DBT_TARGET_PATH", "from_env")
	defer os.Unsetenv("DBT_TARGET_PATH")
	got = ResolveTargetDir(nil)
	if got != "from_env" {
		t.Errorf("expected 'from_env', got %q", got)
	}

	// Test: --target-path takes precedence over DBT_TARGET_PATH
	got = ResolveTargetDir([]string{"run", "--target-path", "from_flag"})
	if got != "from_flag" {
		t.Errorf("expected 'from_flag', got %q", got)
	}

	// Test: SYNQ_TARGET_DIR takes highest precedence
	os.Setenv("SYNQ_TARGET_DIR", "from_synq")
	defer os.Unsetenv("SYNQ_TARGET_DIR")
	got = ResolveTargetDir([]string{"run", "--target-path", "from_flag"})
	if got != "from_synq" {
		t.Errorf("expected 'from_synq', got %q", got)
	}
}

func TestResolveTargetDir_Default(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	os.Unsetenv("SYNQ_TARGET_DIR")
	os.Unsetenv("DBT_TARGET_PATH")

	got := ResolveTargetDir(nil)
	if got != "target" {
		t.Errorf("expected 'target' default, got %q", got)
	}
}
