package synq

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"time"

	dbtv1 "github.com/getsynq/cloud/api/clients/dbt/v1"
	v1 "github.com/getsynq/cloud/api/clients/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	logger = logrus.WithField("app", "synq-dbt")
)

type Api struct {
	client dbtv1.DbtServiceClient
}

func NewApi(url string) (*Api, error) {
	client, err := createDbtServiceClient(url)
	if err != nil {
		return nil, err
	}

	return &Api{
		client: client,
	}, nil
}

func (api *Api) SendRequest(ctx context.Context, dbtArtifacts *v1.DbtResult) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	_, err := api.client.PostDbtResult(timeoutCtx, &dbtv1.PostDbtResultRequest{
		DbtResult: dbtArtifacts,
	})

	return err
}

//
// HELPERS
//

func createDbtServiceClient(url string) (dbtv1.DbtServiceClient, error) {
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

	conn, err := grpc.Dial(url, opts...)
	if err != nil {
		return nil, err
	}

	return dbtv1.NewDbtServiceClient(conn), nil
}
