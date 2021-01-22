package token

import (
	"context"
	"crypto/tls"
	_ "github.com/jeremyary/observability-operator/api/v1"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Types implementing AuthTokenFetcher can retrieve an auth token for
// Observatorium
type AuthTokenFetcher interface {
	Fetch(config *v1.ObservatoriumConfig, oldToken string) (string, int64, error)
}

// Default empty token fetcher
type NilTokenFetcher struct{}

// Fetches auth tokens from dex
type DexTokenFetcher struct {
	Client     client.Client
	Context    context.Context
	HttpClient *http.Client
}

// Returns a token fetcher for the given auth type
func GetTokenFetcher(config *v1.ObservatoriumConfig, ctx context.Context, client client.Client) AuthTokenFetcher {
	if config == nil {
		return NewNilTokenFetcher()
	}

	switch config.AuthType {
	case v1.AuthTypeDex:
		return NewDexTokenFetcher(ctx, client)
	default:
		return NewNilTokenFetcher()
	}
}

func NewNilTokenFetcher() AuthTokenFetcher {
	return &NilTokenFetcher{}
}

func (r *NilTokenFetcher) Fetch(*v1.ObservatoriumConfig, string) (string, int64, error) {
	return "", 0, nil
}

func NewDexTokenFetcher(ctx context.Context, client client.Client) AuthTokenFetcher {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	return &DexTokenFetcher{
		Client:     client,
		Context:    ctx,
		HttpClient: httpClient,
	}
}

func (r *DexTokenFetcher) Fetch(config *v1.ObservatoriumConfig, oldToken string) (string, int64, error) {
	return "", 0, nil
}
