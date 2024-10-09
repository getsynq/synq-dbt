package synq

import (
	ingestdbtv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/ingest/dbt/v1/dbtv1grpc"
	ingestdbtv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/ingest/dbt/v1"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net/url"
)

func UploadMetadata(ctx context.Context, output *ingestdbtv1.IngestInvocationRequest, endpoint string, token string) error {
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	oauthTokenSource, err := LongLivedTokenSource(token, parsedEndpoint)
	if err != nil {
		return err
	}
	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: false})
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(oauthTokenSource),
		grpc.WithAuthority(parsedEndpoint.Host),
	}

	conn, err := grpc.DialContext(ctx, grpcEndpoint(parsedEndpoint), opts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	dbtServiceClient := ingestdbtv1grpc.NewDbtServiceClient(conn)
	resp, err := dbtServiceClient.IngestInvocation(ctx, output)
	if err != nil {
		return err
	}
	logrus.Infof("Metadata uploaded successfully: %s", resp.String())
	return nil
}

func grpcEndpoint(endpoint *url.URL) string {
	port := endpoint.Port()
	if port == "" {
		port = "443"
	}
	return fmt.Sprintf("%s:%s", endpoint.Hostname(), port)
}
