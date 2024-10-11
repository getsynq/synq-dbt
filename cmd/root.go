package cmd

import (
	"context"
	"os"
)

func Execute(ctx context.Context) {
	if len(os.Args) > 1 && os.Args[1] == "upload_artifacts" {
		_ = uploadRunCmd.ExecuteContext(ctx)
	} else {
		_ = runCmd.ExecuteContext(ctx)
	}
}
