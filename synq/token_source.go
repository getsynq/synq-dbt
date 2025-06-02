package synq

import (
	"context"
	"net/url"

	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

type TokenSource interface {
	oauth2.TokenSource
	credentials.PerRPCCredentials
}

func LongLivedTokenSource(ctx context.Context, longLivedToken string, apiEndpoint *url.URL) (TokenSource, error) {
	initialToken, err := obtainToken(ctx, apiEndpoint, longLivedToken)
	if err != nil {
		return nil, err
	}

	return oauth.TokenSource{TokenSource: oauth2.ReuseTokenSource(initialToken, &tokenSource{apiEndpoint: apiEndpoint, longLivedToken: longLivedToken})}, nil
}

type tokenSource struct {
	longLivedToken string
	apiEndpoint    *url.URL
}

func (t *tokenSource) Token() (*oauth2.Token, error) {
	return obtainToken(nil, t.apiEndpoint, t.longLivedToken)
}

func obtainToken(ctx context.Context, apiEndpoint *url.URL, longLivedToken string) (*oauth2.Token, error) {

	tokenURL, _ := url.Parse(apiEndpoint.String())
	tokenURL.Path = "/oauth2/token"
	conf := oauth2.Config{
		Endpoint: oauth2.Endpoint{
			TokenURL:  tokenURL.String(),
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
	return conf.PasswordCredentialsToken(ctx, "synq", longLivedToken)
}
