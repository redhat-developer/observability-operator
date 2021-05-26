package token

import (
	"context"
	"fmt"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/api/v1"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/controllers/token"
	"github.com/go-logr/logr"
	errors2 "github.com/pkg/errors"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

const (
	RemoteTokenValue    = "token"
	RemoteTokenLifetime = "lifetime"
)

func GetObservatoriumTokenSecretName(config *v1.ObservatoriumIndex) string {
	return fmt.Sprintf("obs-token-%v", config.Id)
}

func GetObservatoriumPrometheusSecretName(index *v1.RepositoryIndex) string {
	id := index.Config.Prometheus.Observatorium
	config := GetObservatoriumConfig(index, id)
	return GetObservatoriumTokenSecretName(config)
}

func GetObservatoriumPromtailSecretName(index *v1.RepositoryIndex) string {
	id := index.Config.Promtail.Observatorium
	config := GetObservatoriumConfig(index, id)
	return GetObservatoriumTokenSecretName(config)
}

func GetObservatoriumConfig(index *v1.RepositoryIndex, id string) *v1.ObservatoriumIndex {
	if index == nil || index.Config == nil || index.Config.Observatoria == nil {
		return nil
	}

	for _, observatorium := range index.Config.Observatoria {
		if observatorium.Id == id {
			return &observatorium
		}
	}

	return nil
}

func findToken(ctx context.Context, c client.Client, cr *v1.Observability, config *v1.ObservatoriumIndex) (string, int64, error) {
	secretName := GetObservatoriumTokenSecretName(config)

	secret := &v12.Secret{}
	selector := client.ObjectKey{
		Namespace: cr.Namespace,
		Name:      secretName,
	}

	err := c.Get(ctx, selector, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", 0, nil
		}
		return "", 0, err
	}

	token := secret.Data[RemoteTokenValue]
	if token == nil {
		return "", 0, fmt.Errorf("no token found in %v", secretName)
	}

	lifetime := secret.Data[RemoteTokenLifetime]
	if lifetime == nil {
		lifetime = []byte("0")
	}

	lifetimeLeft, err := strconv.ParseInt(string(lifetime), 10, 64)
	return string(token), lifetimeLeft, err
}

func refreshToken(ctx context.Context, c client.Client, config *v1.ObservatoriumIndex, cr *v1.Observability, oldToken string) (string, int64, error) {
	fetcher := token.GetTokenFetcher(config, ctx, c)
	newToken, expires, err := fetcher.Fetch(cr, config, oldToken)
	if err != nil {
		return "", 0, errors2.Wrap(err, fmt.Sprintf("error fetching token for %v", config.Id))
	}
	return newToken, expires, nil
}

func saveToken(ctx context.Context, c client.Client, config *v1.ObservatoriumIndex, cr *v1.Observability, token string, lifetime int64) error {
	secretName := GetObservatoriumTokenSecretName(config)

	secret := &v12.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: cr.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, c, secret, func() error {
		secret.Labels = map[string]string{
			"managed-by": "observability-operator",
			"purpose":    "observatorium-token-secret",
		}
		secret.StringData = map[string]string{
			RemoteTokenValue:    token,
			RemoteTokenLifetime: strconv.FormatInt(lifetime, 10),
		}
		return nil
	})

	if err != nil {
		return errors2.Wrap(err, fmt.Sprintf("error creating token secret for %v", config.Id))
	}

	return nil
}

func TokensExpired(ctx context.Context, c client.Client, cr *v1.Observability) (bool, error) {
	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"managed-by": "observability-operator",
			"purpose":    "observatorium-token-secret",
		}),
		Namespace: cr.Namespace,
	}

	list := &v12.SecretList{}
	err := c.List(ctx, list, opts)
	if err != nil {
		return false, errors2.Wrap(err, "error checking token secrets for expiration")
	}

	for _, secret := range list.Items {
		lifetime := secret.Data[RemoteTokenLifetime]
		if lifetime == nil {
			return true, nil
		}

		expires, err := strconv.ParseInt(string(lifetime), 10, 64)
		if err != nil {
			return false, errors2.Wrap(err, fmt.Sprintf("error parsing token lifetime for secret %v", secret.Name))
		}

		if token.AuthTokenExpires(expires) {
			return true, nil
		}
	}

	return false, nil
}

func ReconcileObservatoria(log logr.Logger, ctx context.Context, c client.Client, cr *v1.Observability, index *v1.RepositoryIndex) error {
	if index == nil || index.Config == nil || index.Config.Observatoria == nil {
		return nil
	}

	if cr.ObservatoriumDisabled() {
		return nil
	}

	for _, observatorium := range index.Config.Observatoria {
		t, lifetime, err := findToken(ctx, c, cr, &observatorium)
		if err != nil {
			return errors2.Wrap(err, fmt.Sprintf("error checking existing observatorium token for %v", observatorium.Id))
		}

		// No token yet?
		if t == "" || token.AuthTokenExpires(lifetime) {
			t, lifetime, err := refreshToken(ctx, c, &observatorium, cr, t)
			if err != nil {
				log.Error(err, fmt.Sprintf("error fetching token for observatorium %v", observatorium.Id))
				continue
			}

			err = saveToken(ctx, c, &observatorium, cr, t, lifetime)
			if err != nil {
				log.Error(err, fmt.Sprintf("error storing token for observatorium %v", observatorium.Id))
				continue
			}
		}

	}

	return nil
}
