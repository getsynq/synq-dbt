package synq

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	dbtv1 "github.com/getsynq/cloud/api/clients/dbt/v1"
	v1 "github.com/getsynq/cloud/api/clients/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func uploadArtifactsToSYNQ(ctx context.Context, dbtResult *v1.DbtResult, token, url string) error {

	client, err := createLegacyDbtServiceClient(ctx, url)
	if err != nil {
		return err
	}

	dbtResult.Token = token

	_, err = client.PostDbtResult(ctx, &dbtv1.PostDbtResultRequest{
		DbtResult: dbtResult,
	})
	if err != nil {
		return err
	}

	logrus.Infof("synq-dbt upload successful (legacy)")

	return nil
}

func createLegacyDbtServiceClient(ctx context.Context, url string) (dbtv1.DbtServiceClient, error) {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	creds := credentials.NewTLS(&tls.Config{
		RootCAs: certPool,
	})

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	conn, err := grpc.DialContext(ctx, url, opts...)
	if err != nil {
		return nil, err
	}

	return dbtv1.NewDbtServiceClient(conn), nil
}
