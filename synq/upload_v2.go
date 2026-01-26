package synq

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"

	ingestdbtv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/ingest/dbt/v1/dbtv1grpc"
	ingestdbtv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/ingest/dbt/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func uploadArtifactsToSYNQ(ctx context.Context, request *ingestdbtv1.IngestInvocationRequest, token string, synqApiEndpoint string) error {
	if request == nil || token == "" {
		return nil
	}

	parsedEndpoint, err := url.Parse(synqApiEndpoint)
	if err != nil {
		return err
	}

	oauthTokenSource, err := LongLivedTokenSource(ctx, token, parsedEndpoint)
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

	resp, err := dbtServiceClient.IngestInvocation(ctx, request)
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
