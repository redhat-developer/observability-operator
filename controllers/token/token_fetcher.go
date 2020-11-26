package token

import (
	"context"
	"encoding/json"
	"fmt"
	_ "github.com/jeremyary/observability-operator/api/v1"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"io/ioutil"
	v12 "k8s.io/api/core/v1"
	"net/http"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// Types implementing AuthTokenFetcher can retrieve an auth token for
// Observatorium
type AuthTokenFetcher interface {
	Fetch(cr *v1.Observability, oldToken string) (string, int64, error)
}

// Default empty token fetcher
type NilTokenFetcher struct{}

// Fetches auth tokens from dex
type DexTokenFetcher struct {
	Client  client.Client
	Context context.Context
}

// Returns a token fetcher for the given auth type
func GetTokenFetcher(cr *v1.Observability, ctx context.Context, client client.Client) AuthTokenFetcher {
	if cr.Spec.Observatorium == nil {
		return NewNilTokenFetcher()
	}

	switch cr.Spec.Observatorium.AuthType {
	case v1.AuthTypeDex:
		return NewDexTokenFetcher(ctx, client)
	default:
		return NewNilTokenFetcher()
	}
}

func NewNilTokenFetcher() AuthTokenFetcher {
	return &NilTokenFetcher{}
}

func (r *NilTokenFetcher) Fetch(*v1.Observability, string) (string, int64, error) {
	return "", 0, nil
}

func NewDexTokenFetcher(ctx context.Context, client client.Client) AuthTokenFetcher {
	return &DexTokenFetcher{
		Client:  client,
		Context: ctx,
	}
}

func (r *DexTokenFetcher) Fetch(cr *v1.Observability, oldToken string) (string, int64, error) {
	// No config, no token
	if cr.Spec.Observatorium.AuthDex == nil {
		return oldToken, 0, nil
	}

	if cr.Status.TokenExpires > 0 {
		// Refresh token a little bit in advance
		now := time.Now().Add(time.Minute * 5)
		expiry := time.Unix(cr.Status.TokenExpires, 0)

		// Is it really time for renewal?
		if !now.After(expiry) {
			return oldToken, cr.Status.TokenExpires, nil
		}
	}

	// Get credential secret
	secret := &v12.Secret{}
	selector := client.ObjectKey{
		Namespace: cr.Spec.Observatorium.AuthDex.CredentialSecretNamespace,
		Name:      cr.Spec.Observatorium.AuthDex.CredentialSecretName,
	}

	err := r.Client.Get(r.Context, selector, secret)
	if err != nil {
		return oldToken, cr.Status.TokenExpires, err
	}

	tokenEndpoint := fmt.Sprintf("%s/dex/token", cr.Spec.Observatorium.AuthDex.Url)
	formData := url.Values{
		"grant_type":    {"password"},
		"username":      {string(secret.Data["username"])},
		"password":      {string(secret.Data["password"])},
		"client_id":     {cr.Spec.Observatorium.Tenant},
		"client_secret": {string(secret.Data["secret"])},
		"scope":         {"openid email"},
	}

	resp, err := http.PostForm(tokenEndpoint, formData)
	if err != nil {
		return oldToken, cr.Status.TokenExpires, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return oldToken, cr.Status.TokenExpires, err
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
