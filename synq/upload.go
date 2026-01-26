package synq

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	ingestdbtv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/ingest/dbt/v1/dbtv1grpc"
	ingestdbtv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/ingest/dbt/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func UploadArtifacts(ctx context.Context, request *ingestdbtv1.IngestInvocationRequest, token string, targetDirectory string) {
	if request == nil || token == "" {
		return
	}

	endpoint := "https://developer.synq.io/"
	if envEndpoint, ok := os.LookupEnv("SYNQ_API_ENDPOINT"); ok {
		endpoint = envEndpoint
	}

	logrus.Infof("synq-dbt processing `%s`, uploading to `%s`", targetDirectory, endpoint)

	const (
		uploadTimeout = 30 * time.Second
		maxRetries    = 3
	)
	retryDelays := []time.Duration{5 * time.Second, 10 * time.Second, 15 * time.Second}

	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		timeoutCtx, cancel := context.WithTimeout(ctx, uploadTimeout)
		err = ingestInvocation(timeoutCtx, request, token, endpoint)
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
			logrus.Infof("synq-dbt retrying upload in %v...", retryDelays[attempt])
			time.Sleep(retryDelays[attempt])
		}
	}

	logrus.Errorf("synq-dbt upload failed after %d attempts: %s", maxRetries+1, err.Error())
}

func ingestInvocation(ctx context.Context, request *ingestdbtv1.IngestInvocationRequest, token, endpoint string) error {
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	oauthTokenSource, err := LongLivedTokenSource(ctx, token, parsedEndpoint)
	if err != nil {
		return err
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: false})),
		grpc.WithPerRPCCredentials(oauthTokenSource),
		grpc.WithAuthority(parsedEndpoint.Host),
	}

	conn, err := grpc.NewClient(grpcEndpoint(parsedEndpoint), opts...)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	resp, err := ingestdbtv1grpc.NewDbtServiceClient(conn).IngestInvocation(ctx, request)
	if err != nil {
		return err
	}
	logrus.Printf("metadata uploaded successfully: %s", resp.String())
	return nil
}

func grpcEndpoint(endpoint *url.URL) string {
	port := endpoint.Port()
	if port == "" {
		port = "443"
	}
	return fmt.Sprintf("%s:%s", endpoint.Hostname(), port)
}
