package configuration

import (
	"context"
	"fmt"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/model"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/utils"
	"github.com/ghodss/yaml"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) reconcileAlertmanager(ctx context.Context, cr *v1.Observability) error {
	alertmanager := model.GetAlertmanagerCr(cr)
	configSecretName := model.GetAlertmanagerSecretName(cr)
	proxySecret := model.GetAlertmanagerProxySecret(cr)
	sa := model.GetAlertmanagerServiceAccount(cr)

	route := model.GetAlertmanagerRoute(cr)
	selector := client.ObjectKey{
		Namespace: route.Namespace,
		Name:      route.Name,
	}

	err := r.client.Get(ctx, selector, route)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	host := ""
	if utils.IsRouteReady(route) {
		host = route.Spec.Host
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, alertmanager, func() error {
		alertmanager.Spec.ConfigSecret = configSecretName
		alertmanager.Spec.ListenLocal = true
		alertmanager.Spec.ExternalURL = fmt.Sprintf("https://%v", host)
		alertmanager.Spec.ServiceAccountName = sa.Name
		alertmanager.Spec.Secrets = []string{
			proxySecret.Name,
			"alertmanager-k8s-tls",
		}
		alertmanager.Spec.PriorityClassName = model.ObservabilityPriorityClassName
		alertmanager.Spec.Containers = []v12.Container{
			{
				Name:  "oauth-proxy",
				Image: "quay.io/openshift/origin-oauth-proxy:4.2",
				Args: []string{
					"-provider=openshift",
					"-https-address=:9091",
					"-http-address=",
					"-email-domain=*",
					"-upstream=http://localhost:9093",
					"-openshift-sar={\"resource\": \"namespaces\", \"verb\": \"get\"}",
					"-openshift-delegate-urls={\"/\": {\"resource\": \"namespaces\", \"verb\": \"get\"}}",
					"-tls-cert=/etc/tls/private/tls.crt",
					"-tls-key=/etc/tls/private/tls.key",
					"-client-secret-file=/var/run/secrets/kubernetes.io/serviceaccount/token",
					"-cookie-secret-file=/etc/proxy/secrets/session_secret",
					fmt.Sprintf("-openshift-service-account=%v", sa.Name),
					"-openshift-ca=/etc/pki/tls/cert.pem",
					"-openshift-ca=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
					"-skip-auth-regex=^/metrics",
				},
				Ports: []v12.ContainerPort{
					{
						Name:          "proxy",
						ContainerPort: 9091,
					},
				},
				Env: []v12.EnvVar{
					{
						Name: "HTTP_PROXY",
					},
					{
						Name: "HTTPS_PROXY",
					},
					{
						Name: "NO_PROXY",
					},
				},
				VolumeMounts: []v12.VolumeMount{
					{
						Name:      "secret-alertmanager-k8s-tls",
						MountPath: "/etc/tls/private",
					},
					{
						Name:      fmt.Sprintf("secret-%v", proxySecret.Name),
						MountPath: "/etc/proxy/secrets",
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return err
	}

	return err

}

func (r *Reconciler) reconcileAlertmanagerSecret(ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex) error {
	root := &v1.AlertmanagerConfigRoute{
		Receiver: "default-receiver",
		Routes:   []v1.AlertmanagerConfigRoute{},
	}

	config := v1.AlertmanagerConfigRoot{
		Global: &v1.AlertmanagerConfigGlobal{
			ResolveTimeout: "5m",
		},
		Route: root,
		Receivers: []v1.AlertmanagerConfigReceiver{
			{
				Name: "default-receiver",
			},
		},
	}

	for _, index := range indexes {
		if index.Config == nil || index.Config.Alertmanager == nil {
			continue
		}

		if !cr.PagerDutyDisabled() {
			pagerDutySecret, err := r.getPagerDutySecret(ctx, cr, index.Config.Alertmanager)
			if err != nil {
				r.logger.Error(err, fmt.Sprintf("pagerduty secret %v not found", index.Config.Alertmanager.PagerDutySecretName))
				continue
			}

			pagerDutyReceiver := fmt.Sprintf("%s-%s", index.Id, "pagerduty")

			config.Receivers = append(config.Receivers, v1.AlertmanagerConfigReceiver{
				Name: pagerDutyReceiver,
				PagerDutyConfigs: []v1.PagerDutyConfig{
					{
						ServiceKey: string(pagerDutySecret),
					},
				},
			})

			root.Routes = append(root.Routes, v1.AlertmanagerConfigRoute{
				Receiver: pagerDutyReceiver,
				Match: map[string]string{
					"severity":                  "critical",
					PrometheusRuleIdentifierKey: index.Id,
				},
			})
		}

		if !cr.DeadMansSnitchDisabled() {
			deadmansSnitchUrl, err := r.getDeadMansSnitchUrl(ctx, cr, index.Config.Alertmanager)
			if err != nil {
				r.logger.Error(err, fmt.Sprintf("deadmanssnitch secret %v not found", index.Config.Alertmanager.DeadmansSnitchSecretName))
				continue
			}

			deadMansSnitchReceiver := fmt.Sprintf("%s-%s", index.Id, "deadmanssnitch")

			config.Receivers = append(config.Receivers, v1.AlertmanagerConfigReceiver{
				Name: deadMansSnitchReceiver,
				WebhookConfigs: []v1.WebhookConfig{
					{
						Url: string(deadmansSnitchUrl),
					},
				},
			})

			root.Routes = append(root.Routes, v1.AlertmanagerConfigRoute{
				Receiver:       deadMansSnitchReceiver,
				RepeatInterval: "5m",
				Match: map[string]string{
					"alertname":                 "DeadMansSwitch",
					PrometheusRuleIdentifierKey: index.Id,
				},
			})
		}
	}

	configBytes, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}

	secret := model.GetAlertmanagerSecret(cr)

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, secret, func() error {
		secret.Type = v12.SecretTypeOpaque
		secret.StringData = map[string]string{
			"alertmanager.yaml": string(configBytes),
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) getPagerDutySecret(ctx context.Context, cr *v1.Observability, config *v1.AlertmanagerIndex) ([]byte, error) {
	if config == nil {
		return []byte("dummy"), nil
	}

	if config.PagerDutySecretName == "" {
		return []byte("dummy"), nil
	}

	ns := cr.Namespace
	if config.PagerDutySecretNamespace != "" {
		ns = config.PagerDutySecretNamespace
	}

	pagerdutySecret := &v12.Secret{}
	selector := client.ObjectKey{
		Namespace: ns,
		Name:      config.PagerDutySecretName,
	}
	err := r.client.Get(ctx, selector, pagerdutySecret)
	if err != nil {
		return nil, err
	}

	var secret []byte
	if len(pagerdutySecret.Data["PAGERDUTY_KEY"]) != 0 {
		secret = pagerdutySecret.Data["PAGERDUTY_KEY"]
	} else if len(pagerdutySecret.Data["serviceKey"]) != 0 {
		secret = pagerdutySecret.Data["serviceKey"]
	}

	return secret, nil
}

func (r *Reconciler) getDeadMansSnitchUrl(ctx context.Context, cr *v1.Observability, config *v1.AlertmanagerIndex) ([]byte, error) {
	if config == nil {
		return []byte("http://dummy"), nil
	}

	if config.DeadmansSnitchSecretName == "" {
		return []byte("http://dummy"), nil
	}

	ns := cr.Namespace
	if config.DeadmansSnitchSecretNamespace != "" {
		ns = config.DeadmansSnitchSecretNamespace
	}

	dmsSecret := &v12.Secret{}
	selector := client.ObjectKey{
		Namespace: ns,
		Name:      config.DeadmansSnitchSecretName,
	}
	err := r.client.Get(ctx, selector, dmsSecret)
	if err != nil {
		return nil, err
	}

	var url []byte
	if len(dmsSecret.Data["SNITCH_URL"]) != 0 {
		url = dmsSecret.Data["SNITCH_URL"]
	} else if len(dmsSecret.Data["url"]) != 0 {
		url = dmsSecret.Data["url"]
	}

	return url, nil
}
