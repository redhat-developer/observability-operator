package token

import (
	"context"
	"fmt"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/token"
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
	RemoteTokenValue                  = "token"
	RemoteTokenLifetime               = "lifetime"
	ObservatoriumSecretKeyDexPassword = "dexPassword"
	ObservatoriumSecretKeyDexSecret   = "dexSecret"
	ObservatoriumSecretKeyDexUsername = "dexUsername"
	ObservatoriumSecretKeyDexUrl      = "dexUrl"
	ObservatoriumSecretKeyTenant      = "tenant"
	ObservatoriumSecretKeyGateway     = "gateway"
	ObservatoriumSecretKeyAuthType    = "authType"

	ObservatoriumSecretKeyRedhatSsoUrl   = "redHatSsoAuthServerUrl"
	ObservatoriumSecretKeyRedhatSsoRealm = "redHatSsoRealm"
	ObservatoriumSecretKeyMetricsClient  = "metricsClientId"
	ObservatoriumSecretKeyMetricsSecret  = "metricsSecret"
	ObservatoriumSecretKeyLogsClient     = "logsClientId"
	ObservatoriumSecretKeyLogsSecret     = "logsSecret"
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

func assignFromSecret(ctx context.Context, c client.Client, cr *v1.Observability, index *v1.ObservatoriumIndex) error {
	targetSecret := &v12.Secret{}
	selector := client.ObjectKey{
		Namespace: cr.Namespace,
		Name:      index.SecretName,
	}

	err := c.Get(ctx, selector, targetSecret)
	if err != nil {
		return err
	}

	if index.AuthType == "" {
		index.AuthType = v1.ObservabilityAuthType(targetSecret.Data[ObservatoriumSecretKeyAuthType])
	}

	if index.Gateway == "" {
		index.Gateway = string(targetSecret.Data[ObservatoriumSecretKeyGateway])
	}

	if index.Tenant == "" {
		index.Tenant = string(targetSecret.Data[ObservatoriumSecretKeyTenant])
	}

	switch index.AuthType {
	case v1.AuthTypeDex:
		// This not being nil means the configuration was given via the repository. This is still
		// supported for backwards compatibility, however the prefered way is by secret.
		if index.DexConfig != nil {
			return assignFromSecret(ctx, c, cr, index)
		}

		// Dex configuration is part of the config secret created by the KAS Fleet Manager
		index.DexConfig = new(v1.DexConfig)
		index.DexConfig.Url = string(targetSecret.Data[ObservatoriumSecretKeyDexUrl])
		index.DexConfig.Secret = string(targetSecret.Data[ObservatoriumSecretKeyDexSecret])
		index.DexConfig.Password = string(targetSecret.Data[ObservatoriumSecretKeyDexPassword])
		index.DexConfig.Username = string(targetSecret.Data[ObservatoriumSecretKeyDexUsername])
	case v1.AuthTypeRedhat:
		// RedHat SSO credentials are required to be part of the config secret created by the
		// KAS Fleet Manager. This auth type is not supported in the config repository.
		index.RedhatSsoConfig = new(v1.RedhatSsoConfig)
		index.RedhatSsoConfig.Url = string(targetSecret.Data[ObservatoriumSecretKeyRedhatSsoUrl])
		index.RedhatSsoConfig.Realm = string(targetSecret.Data[ObservatoriumSecretKeyRedhatSsoRealm])
		index.RedhatSsoConfig.MetricsSecret = string(targetSecret.Data[ObservatoriumSecretKeyMetricsSecret])
		index.RedhatSsoConfig.MetricsClient = string(targetSecret.Data[ObservatoriumSecretKeyMetricsClient])
		index.RedhatSsoConfig.LogsSecret = string(targetSecret.Data[ObservatoriumSecretKeyLogsSecret])
		index.RedhatSsoConfig.LogsClient = string(targetSecret.Data[ObservatoriumSecretKeyLogsClient])
	default:
		return errors2.New(fmt.Sprintf("unknown auth type %v", index.AuthType))
	}

	return nil
}

func ReconcileObservatoria(log logr.Logger, ctx context.Context, c client.Client, cr *v1.Observability, index *v1.RepositoryIndex) error {
	if index == nil || index.Config == nil || index.Config.Observatoria == nil {
		return nil
	}

	if cr.ObservatoriumDisabled() {
		return nil
	}

	var transformed []v1.ObservatoriumIndex

	for _, observatorium := range index.Config.Observatoria {
		if observatorium.SecretName != "" {
			err := assignFromSecret(ctx, c, cr, &observatorium)
			if err != nil {
				log.Error(err, fmt.Sprintf("error finding observatorium secret %v", observatorium.SecretName))
				return err
			}
		}

		if observatorium.AuthType == v1.AuthTypeDex && observatorium.DexConfig != nil && observatorium.DexConfig.CredentialSecretName != "" {
			// By default look for the dex secret in the same namespace as the CR
			namespace := cr.Namespace
			if observatorium.DexConfig.CredentialSecretNamespace != "" {
				namespace = observatorium.DexConfig.CredentialSecretNamespace
			}

			// Get credential secret
			secret := &v12.Secret{}
			selector := client.ObjectKey{
				Namespace: namespace,
				Name:      observatorium.DexConfig.CredentialSecretName,
			}

			err := c.Get(ctx, selector, secret)
			if err != nil {
				return err
			}

			if secret.Data["username"] != nil {
				observatorium.DexConfig.Username = string(secret.Data["username"])
			} else {
				observatorium.DexConfig.Username = string(secret.Data["dexUsername"])
			}

			if secret.Data["password"] != nil {
				observatorium.DexConfig.Password = string(secret.Data["password"])
			} else {
				observatorium.DexConfig.Password = string(secret.Data["dexPassword"])
			}

			if secret.Data["secret"] != nil {
				observatorium.DexConfig.Secret = string(secret.Data["secret"])
			} else {
				observatorium.DexConfig.Secret = string(secret.Data["dexSecret"])
			}
		}

		copy := v1.ObservatoriumIndex{}
		observatorium.DeepCopyInto(&copy)
		transformed = append(transformed, copy)

		// No token fetching required if we are using RedHat SSO. The token-refresher proxy
		// is taking care of that for us
		if observatorium.AuthType == v1.AuthTypeRedhat {
			continue
		}

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

	index.Config.Observatoria = transformed
	return nil
}
