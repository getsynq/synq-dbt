package synq

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	v1 "github.com/getsynq/cloud/api/clients/v1"
	"github.com/sirupsen/logrus"
)

func UploadArtifacts(ctx context.Context, dbtResult *v1.DbtResult, token string, targetDirectory string) {
	synqV1ApiEndpoint := "dbtapi.synq.io:443"
	if envEndpoint, ok := os.LookupEnv("SYNQ_UPLOAD_URL"); ok {
		synqV1ApiEndpoint = envEndpoint
	}
	synqV2ApiEndpoint := "https://developer.synq.io/"
	if envEndpoint, ok := os.LookupEnv("SYNQ_API_ENDPOINT"); ok {
		synqV2ApiEndpoint = envEndpoint
	}

	uploadTimeout := time.Second * 30
	maxRetries := 3
	retryDelays := []time.Duration{5 * time.Second, 10 * time.Second, 15 * time.Second}

	useSYNQApiV2, _ := strconv.ParseBool(os.Getenv("SYNQ_API_V2"))
	useSYNQApiV2 = useSYNQApiV2 || strings.HasPrefix(token, "st-")

	if useSYNQApiV2 {
		logrus.Infof("synq-dbt processing `%s`, uploading to `%s` using v2 API", targetDirectory, synqV2ApiEndpoint)
	} else {
		logrus.Infof("synq-dbt processing `%s`, uploading to `%s` using legacy API", targetDirectory, synqV1ApiEndpoint)
	}

	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		timeoutCtx, cancel := context.WithTimeout(ctx, uploadTimeout)

		if useSYNQApiV2 {
			err = uploadArtifactsToSYNQV2(timeoutCtx, dbtResult, token, synqV2ApiEndpoint)
		} else {
			err = uploadArtifactsToSYNQ(timeoutCtx, dbtResult, token, synqV1ApiEndpoint)
		}

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
