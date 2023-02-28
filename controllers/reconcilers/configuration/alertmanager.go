package configuration

import (
	"context"
	"fmt"
	"strings"

	goyaml "github.com/goccy/go-yaml"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	"github.com/redhat-developer/observability-operator/v4/controllers/model"
	"github.com/redhat-developer/observability-operator/v4/controllers/utils"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) reconcileAlertmanager(ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex) error {
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
		alertmanager.Spec = prometheusv1.AlertmanagerSpec{
			PodMetadata: &prometheusv1.EmbeddedObjectMetadata{
				Annotations: map[string]string{
					"cluster-autoscaler.kubernetes.io/safe-to-evict": "true",
				},
			},
			ConfigSecret:       configSecretName,
			ListenLocal:        true,
			ExternalURL:        fmt.Sprintf("https://%v", host),
			ServiceAccountName: sa.Name,
			Secrets: []string{
				proxySecret.Name,
				"alertmanager-k8s-tls",
			},
			PriorityClassName: model.ObservabilityPriorityClassName,
			Containers: []v12.Container{
				{
					Name:  "oauth-proxy",
					Image: GetOriginOauthProxyImage(cr),
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
			},
			Version:   model.GetAlertmanagerVersion(cr),
			Resources: *model.GetAlertmanagerResourceRequirement(cr),
		}
		alertmanager.Spec.Version = model.GetAlertmanagerVersion(cr)
		alertmanager.Spec.Resources = *model.GetAlertmanagerResourceRequirement(cr)
		if cr.Spec.Storage != nil && cr.Spec.Storage.AlertManagerStorageSpec != nil {
			alertManagerStorageSpec, err := getAlertManagerStorageSpecHelper(cr, indexes)
			if err != nil {
				return err
			}
			alertmanager.Spec.Storage = alertManagerStorageSpec
		}
		return nil
	})
	if err != nil {
		return err
	}

	return err

}

// construct Alertmanager storage spec with either default or override value from resources
func getAlertManagerStorageSpecHelper(cr *v1.Observability, indexes []v1.RepositoryIndex) (*prometheusv1.StorageSpec, error) {
	alertManagerStorageSpec := cr.Spec.Storage.AlertManagerStorageSpec
	customStorageSize := model.GetAlertmanagerStorageSize(cr, indexes)
	if customStorageSize == "" {
		return alertManagerStorageSpec, nil
	}
	_, err := resource.ParseQuantity(customStorageSize) //check if resources value is valid
	return cr.Spec.Storage.AlertManagerStorageSpec, err
}

func (r *Reconciler) reconcileAlertmanagerSecret(ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex) error {
	root := &v1.AlertmanagerConfigRoute{
		Receiver:       "default-receiver",
		RepeatInterval: "12h",
		Routes:         []v1.AlertmanagerConfigRoute{},
	}

	globalConfig, err := r.createGlobalConfig(ctx, cr, indexes)

	if err != nil {
		return err
	}

	config := v1.AlertmanagerConfigRoot{
		Global: globalConfig,
		Route:  root,
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

		if !cr.SmtpDisabled() && len(index.Config.Alertmanager.SmtpToEmailAddress) > 0 && index.Config.Alertmanager.SmtpFromEmailAddress != "" {

			smtpReceiver := fmt.Sprintf("%s-%s", index.Id, "smtp")

			toEmailAddress := ""
			if len(index.Config.Alertmanager.SmtpToEmailAddress) > 1 {
				toEmailAddress = strings.Join(index.Config.Alertmanager.SmtpToEmailAddress, ",")
			} else {
				toEmailAddress = index.Config.Alertmanager.SmtpToEmailAddress[0]
			}

			config.Receivers = append(config.Receivers, v1.AlertmanagerConfigReceiver{
				Name: smtpReceiver,
				EmailConfig: []v1.EmailConfig{
					{
						SendResolved: true,
						To:           toEmailAddress,
					},
				},
			})

			root.Routes = append(root.Routes, v1.AlertmanagerConfigRoute{
				Receiver: smtpReceiver,
				Match: map[string]string{
					"severity":                  "warning",
					PrometheusRuleIdentifierKey: index.Id,
				},
			})
		}
	}

	configBytes, err := goyaml.Marshal(&config)
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

func (r *Reconciler) getSmtpSecret(ctx context.Context, cr *v1.Observability, config *v1.AlertmanagerIndex) (map[string][]byte, error) {

	secrets := make(map[string][]byte)

	if config == nil {
		return secrets, nil
	}

	if config.SmtpSecretName == "" {
		return secrets, nil
	}

	ns := cr.Namespace
	if config.SmtpSecretNamespace != "" {
		ns = config.SmtpSecretNamespace
	}

	SmtpSecret := &v12.Secret{}
	selector := client.ObjectKey{
		Namespace: ns,
		Name:      config.SmtpSecretName,
	}
	err := r.client.Get(ctx, selector, SmtpSecret)
	if err != nil {
		return nil, err
	}

	if len(SmtpSecret.Data["password"]) != 0 {
		secrets["password"] = SmtpSecret.Data["password"]
	}

	if len(SmtpSecret.Data["username"]) != 0 {
		secrets["username"] = SmtpSecret.Data["username"]
	}

	if len(SmtpSecret.Data["host"]) != 0 {
		secrets["host"] = SmtpSecret.Data["host"]
	}

	if len(SmtpSecret.Data["port"]) != 0 {
		secrets["port"] = SmtpSecret.Data["port"]
	}

	return secrets, nil
}

func (r *Reconciler) createGlobalConfig(ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex) (*v1.AlertmanagerConfigGlobal, error) {

	globalConfig := &v1.AlertmanagerConfigGlobal{
		ResolveTimeout: "5m",
	}

	if (!cr.SmtpDisabled() && len(indexes[0].Config.Alertmanager.SmtpToEmailAddress) == 0) || (!cr.SmtpDisabled() && indexes[0].Config.Alertmanager.SmtpFromEmailAddress == "") {
		r.logger.Info("both the to and from email address in the index.json file need to be set when smtp is enabled")
	} else if !cr.SmtpDisabled() && len(indexes[0].Config.Alertmanager.SmtpToEmailAddress) > 0 && indexes[0].Config.Alertmanager.SmtpFromEmailAddress != "" {

		smtpSecret, err := r.getSmtpSecret(ctx, cr, indexes[0].Config.Alertmanager)

		if err != nil {
			r.logger.Error(err, fmt.Sprintf("smtp secret %v not found", indexes[0].Config.Alertmanager.SmtpSecretName))
			return nil, err
		}

		// If the secret is empty provide it with dummy values
		if len(smtpSecret) == 0 {
			smtpSecret["password"] = []byte("dummy")
			smtpSecret["username"] = []byte("dummy")
			smtpSecret["host"] = []byte("dummy")
			smtpSecret["port"] = []byte("dummy")
		}

		globalConfig = &v1.AlertmanagerConfigGlobal{
			ResolveTimeout:   "5m",
			SmtpAuthUserName: string(smtpSecret["username"]),
			SmtpAuthPassword: string(smtpSecret["password"]),
			SmtpSmartHost:    fmt.Sprintf("%s:%s", string(smtpSecret["host"]), string(smtpSecret["port"])),
			SmtpFrom:         indexes[0].Config.Alertmanager.SmtpFromEmailAddress,
		}

	} else {
		globalConfig = &v1.AlertmanagerConfigGlobal{
			ResolveTimeout: "5m",
		}
	}
	return globalConfig, nil
}
