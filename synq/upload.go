package synq

import (
	"context"
	"errors"
	"os"
	"time"

	ingestdbtv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/ingest/dbt/v1"
	"github.com/sirupsen/logrus"
)

func UploadArtifacts(ctx context.Context, request *ingestdbtv1.IngestInvocationRequest, token string, targetDirectory string) {
	synqApiEndpoint := "https://developer.synq.io/"
	if envEndpoint, ok := os.LookupEnv("SYNQ_API_ENDPOINT"); ok {
		synqApiEndpoint = envEndpoint
	}

	uploadTimeout := time.Second * 30
	maxRetries := 3
	retryDelays := []time.Duration{5 * time.Second, 10 * time.Second, 15 * time.Second}

	logrus.Infof("synq-dbt processing `%s`, uploading to `%s`", targetDirectory, synqApiEndpoint)

	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		timeoutCtx, cancel := context.WithTimeout(ctx, uploadTimeout)

		err = uploadArtifactsToSYNQ(timeoutCtx, request, token, synqApiEndpoint)

		cancel()

		if err == nil {
			logrus.Info("synq-dbt processing and upload successfully finished")
			return
		}

		attemptNumber := attempt + 1
		totalAttempts := maxRetries + 1
		if errors.Is(err, context.DeadlineExceeded) {
			logrus.Warnf("synq-dbt upload timed out after %s on attempt %d/%d", uploadTimeout, attemptNumber, totalAttempts)
		} else {
			logrus.Warnf("synq-dbt upload failed on attempt %d/%d: %s", attemptNumber, totalAttempts, err.Error())
		}

		if attempt < maxRetries {
			delay := retryDelays[attempt]
			logrus.Infof("synq-dbt retrying upload in %v...", delay)
			time.Sleep(delay)
		}
	}

	if err != nil {
		logrus.Errorf("synq-dbt upload failed after %d attempts: %s", maxRetries+1, err.Error())
	} else {
		logrus.Errorf("synq-dbt upload failed after %d attempts", maxRetries+1)
	}
}
