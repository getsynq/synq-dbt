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
	synqV1ApiEndpoint, ok := os.LookupEnv("SYNQ_UPLOAD_URL")
	if !ok {
		synqV1ApiEndpoint = "dbtapi.synq.io:443"
	}
	synqV2ApiEndpoint := "https://developer.synq.io/"
	if envEndpoint, ok := os.LookupEnv("SYNQ_API_ENDPOINT"); ok {
		synqV2ApiEndpoint = envEndpoint
	}

	uploadTimeout := time.Second * 30

	timeoutCtx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()

	var err error
	useSYNQApiV2, _ := strconv.ParseBool(os.Getenv("SYNQ_API_V2"))
	useSYNQApiV2 = useSYNQApiV2 || strings.HasPrefix(token, "st-")
	if useSYNQApiV2 {
		logrus.Infof("synq-dbt processing `%s`, uploading to `%s` using v2 API", targetDirectory, synqV2ApiEndpoint)
		err = uploadArtifactsToSYNQV2(timeoutCtx, dbtResult, token, synqV2ApiEndpoint)
	} else {
		logrus.Infof("synq-dbt processing `%s`, uploading to `%s` using legacy API", targetDirectory, synqV1ApiEndpoint)
		err = uploadArtifactsToSYNQ(timeoutCtx, dbtResult, token, synqV1ApiEndpoint)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		logrus.Warnf("synq-dbt upload timed out after %s", uploadTimeout)
	}

	if err != nil {
		logrus.Errorf("synq-dbt upload failed: %s", err.Error())
	} else {
		logrus.Info("synq-dbt processing and upload successfully finished")
	}
}
