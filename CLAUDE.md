# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

synq-dbt is a CLI tool that wraps dbt execution and uploads dbt artifacts (manifest.json, run_results.json, catalog.json, sources.json) to SYNQ. It's designed for dbt Core users running dbt on Airflow, Dagster, or similar orchestrators.

## Build Commands

```bash
# Generate version info (required before building)
go generate

# Build for current platform
go build main.go

# Build for specific platforms
GOOS=darwin CGO_ENABLED=0 GOARCH=arm64 go build -o synq-dbt-arm64-darwin main.go
GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -o synq-dbt-amd64-linux main.go

# Run tests
go test -v ./...

# Tidy dependencies
go mod tidy
```

## Architecture

The CLI has two modes of operation:
1. **Wrap mode** (default): Executes dbt with all arguments, captures output, collects artifacts, and uploads to SYNQ
2. **Upload mode** (`synq_upload_artifacts`): Uploads pre-existing artifacts without running dbt

### Package Structure

- **cmd/** - CLI commands using cobra
  - `run.go` - Main wrap mode: executes dbt, collects artifacts, uploads
  - `upload_synq.go` - Upload-only mode for pre-existing artifacts
  - `root.go` - Command router
- **dbt/** - Artifact collection
  - `artifacts.go` - Reads JSON files from target directory
  - `result.go` - `Artifacts` struct holding raw JSON strings
- **synq/** - SYNQ API integration
  - `upload.go` - Upload with retries and gRPC client
  - `request.go` - `RequestBuilder` for constructing `IngestInvocationRequest`
  - `token_source.go` - OAuth2 token management
- **git/** - Git context collection (branch, commit, clone URL)
- **command/** - Subprocess execution helpers
- **build/** - Embedded version info (version.txt, time.txt)

### API

Uses gRPC to communicate with SYNQ via `buf.build/gen/go/getsynq/api`. The `IngestInvocationRequest` contains:
- dbt artifacts (manifest, run_results, sources, catalog)
- Execution metadata (args, exit code, stdout/stderr)
- Environment variables (Airflow context)
- Git context
- Uploader version

### Environment Variables

- `SYNQ_TOKEN` - Required. Must start with `st-`
- `SYNQ_API_ENDPOINT` - Optional. Default: `https://developer.synq.io/`. US region: `https://api.us.synq.io`
- `SYNQ_TARGET_DIR` - Optional. Default: `target`
- `SYNQ_DBT_BIN` - Optional. Default: `dbt`

### Collected Airflow Context

The tool collects these env vars when present: `AIRFLOW_CTX_DAG_OWNER`, `AIRFLOW_CTX_DAG_ID`, `AIRFLOW_CTX_TASK_ID`, `AIRFLOW_CTX_EXECUTION_DATE`, `AIRFLOW_CTX_TRY_NUMBER`, `AIRFLOW_CTX_DAG_RUN_ID`
