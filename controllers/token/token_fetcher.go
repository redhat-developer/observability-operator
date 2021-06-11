package token

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	_ "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	"io/ioutil"
	"net/http"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// Types implementing AuthTokenFetcher can retrieve an auth token for
// Observatorium
type AuthTokenFetcher interface {
	Fetch(cr *v1.Observability, config *v1.ObservatoriumIndex, oldToken string) (string, int64, error)
}

// Default empty token fetcher
type NilTokenFetcher struct{}

// Fetches auth tokens from dex
type DexTokenFetcher struct {
	Client     client.Client
	Context    context.Context
	HttpClient *http.Client
}

func AuthTokenExpires(expires int64) bool {
	if expires > 0 {
		// Refresh the token a little bit in advance
		now := time.Now().Add(time.Hour * 1)
		expiry := time.Unix(expires, 0)

		// Is it really time for renewal?
		return now.After(expiry)
	} else {
		return false
	}
}

// Returns a token fetcher for the given auth type
func GetTokenFetcher(config *v1.ObservatoriumIndex, ctx context.Context, client client.Client) AuthTokenFetcher {
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

func (r *NilTokenFetcher) Fetch(*v1.Observability, *v1.ObservatoriumIndex, string) (string, int64, error) {
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

func (r *DexTokenFetcher) Fetch(cr *v1.Observability, config *v1.ObservatoriumIndex, oldToken string) (string, int64, error) {
	// No config, no token
	if config.DexConfig == nil {
		return oldToken, 0, nil
	}

	tokenEndpoint := fmt.Sprintf("%s/dex/token", config.DexConfig.Url)
	formData := url.Values{
		"grant_type":    {"password"},
		"username":      {config.DexConfig.Username},
		"password":      {config.DexConfig.Password},
		"client_id":     {config.Tenant},
		"client_secret": {config.DexConfig.Secret},
		"scope":         {"openid email"},
	}

	resp, err := r.HttpClient.PostForm(tokenEndpoint, formData)
	if err != nil {
		return oldToken, cr.Status.TokenExpires, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return oldToken, cr.Status.TokenExpires, err
	}
	if resp.StatusCode != http.StatusOK {
		return oldToken, cr.Status.TokenExpires, fmt.Errorf("unexpected response from token endpoint: %v", resp.Status)
	}

	dexResponse := struct {
		AccessToken string `json:"id_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}{}

	err = json.Unmarshal(body, &dexResponse)
	if err != nil {
		return oldToken, cr.Status.TokenExpires, err
	}

	// Remember the expiry date so we can refetch only when needed
	expires := time.Now().Add(time.Second * time.Duration(dexResponse.ExpiresIn)).Unix()
	return dexResponse.AccessToken, expires, nil
}
