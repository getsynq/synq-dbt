package cmd

import (
	"context"
)

func Execute(ctx context.Context) {
	runCmd.SetContext(ctx)
	_ = runCmd.Execute()
}
