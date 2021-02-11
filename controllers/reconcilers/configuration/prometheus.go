package configuration

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/api/v1"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/controllers/model"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/controllers/utils"
	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/ghodss/yaml"
	kv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

func (r *Reconciler) fetchFederationConfigs(indexes []v1.RepositoryIndex) ([]string, error) {
	var result []string

	type federationPatterns struct {
		Match []string `json:"match[]"`
	}

	hasPattern := func(newPattern string) bool {
		for _, pattern := range result {
			if newPattern == pattern {
				return true
			}
		}
		return false
	}

	for _, index := range indexes {
		if index.Config == nil || index.Config.Prometheus == nil || index.Config.Prometheus.Federation == "" {
			continue
		}

		federationConfigUrl := fmt.Sprintf("%s/%s", index.BaseUrl, index.Config.Prometheus.Federation)
		bytes, err := r.fetchResource(federationConfigUrl, index.AccessToken)
		if err != nil {
			return nil, err
		}

		var indexConfig federationPatterns
		err = yaml.Unmarshal(bytes, &indexConfig)
		if err != nil {
			return nil, err
		}

		for _, pattern := range indexConfig.Match {
			if hasPattern(pattern) == false {
				result = append(result, fmt.Sprintf("'%s'", pattern))
			}
		}
	}

	return result, nil
}

// Write the additional scrape config secret, used to federate from openshift-monitoring
// This expects the aggregation of all federation configs across all indexes
func (r *Reconciler) createAdditionalScrapeConfigSecret(cr *v1.Observability, ctx context.Context, patterns []string) error {
	secret := model.GetPrometheusAdditionalScrapeConfig(cr)

	user, password, err := r.getOpenshiftMonitoringCredentials(ctx)
	if err != nil {
		return err
	}

	federationConfig, err := model.GetFederationConfig(user, password, patterns)
	if err != nil {
		return err
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, secret, func() error {
		secret.Type = kv1.SecretTypeOpaque
		secret.StringData = map[string]string{
			"additional-scrape-config.yaml": string(federationConfig),
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) getOpenshiftMonitoringCredentials(ctx context.Context) (string, string, error) {
	secret := &kv1.Secret{}
	selector := client.ObjectKey{
		Namespace: "openshift-monitoring",
		Name:      "grafana-datasources",
	}

	err := r.client.Get(ctx, selector, secret)
	if err != nil {
		return "", "", err
	}

	// It says yaml but it's actually json
	j := secret.Data["prometheus.yaml"]

	type datasource struct {
		BasicAuthUser     string `json:"basicAuthUser"`
		BasicAuthPassword string `json:"basicAuthPassword"`
	}

	type datasources struct {
		Sources []datasource `json:"datasources"`
	}

	ds := &datasources{}
	err = json.Unmarshal(j, ds)
	if err != nil {
		return "", "", err
	}

	return ds.Sources[0].BasicAuthUser, ds.Sources[0].BasicAuthPassword, nil
}

func (r *Reconciler) getTokenSecrets(ctx context.Context, cr *v1.Observability) ([]string, error) {
	list := &kv1.SecretList{}
	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"managed-by": "observability-operator",
			"purpose":    "observatorium-token-secret",
		}),
		Namespace: cr.Namespace,
	}

	err := r.client.List(ctx, list, opts)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, secret := range list.Items {
		result = append(result, secret.Name)
	}

	return result, nil
}

func (r *Reconciler) getRemoteWriteIndex(index v1.RepositoryIndex) (*v1.RemoteWriteIndex, error) {
	patternUrl := fmt.Sprintf("%s/%s", index.BaseUrl, index.Config.Prometheus.RemoteWrite)
	bytes, err := r.fetchResource(patternUrl, index.AccessToken)
	if err != nil {
		return nil, err
	}

	remoteWrite := v1.RemoteWriteIndex{}
	err = json.Unmarshal(bytes, &remoteWrite)
	if err != nil {
		return nil, err
	}
	return &remoteWrite, nil
}

func (r *Reconciler) getRemoteWriteSpec(index v1.RepositoryIndex, secrets []string, remoteWrite *v1.RemoteWriteIndex) (*prometheusv1.RemoteWriteSpec, error) {
	indexToken := fmt.Sprintf("%s-observatorium-credentials", index.Id)
	// sanity check that secret we require for current index exists
	found := false
	for _, secret := range secrets {
		if secret == indexToken {
			found = true
			break
		}
	}
	if !found {
		return nil, errors.NewNotFound(kv1.Resource("secret"), indexToken)
	}

	observatoriumConfig := index.Config.Prometheus.Observatorium
	return &prometheusv1.RemoteWriteSpec{
		URL: fmt.Sprintf("%s/api/metrics/v1/%s/api/v1/receive", observatoriumConfig.Gateway, observatoriumConfig.Tenant),
		WriteRelabelConfigs: []prometheusv1.RelabelConfig{
			{
				SourceLabels: []string{"__name__"},
				Regex:        fmt.Sprintf("(%s)", strings.Join(remoteWrite.Patterns, "|")),
				Action:       "keep",
			},
		},
		BearerTokenFile: fmt.Sprintf("/etc/prometheus/secrets/%s/token", indexToken),
		TLSConfig: &prometheusv1.TLSConfig{
			InsecureSkipVerify: true,
		},
	}, nil
}

func (r *Reconciler) getAlerting(cr *v1.Observability) *prometheusv1.AlertingSpec {
	alertmanager := model.GetAlertmanagerCr(cr)
	alertmanagerService := model.GetAlertmanagerService(cr)

	return &prometheusv1.AlertingSpec{
		Alertmanagers: []prometheusv1.AlertmanagerEndpoints{
			{
				Namespace: cr.Namespace,
				Name:      alertmanager.Name,
				Port:      intstr.FromString("web"),
				Scheme:    "https",
				TLSConfig: &prometheusv1.TLSConfig{
					CAFile:     "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt",
					ServerName: fmt.Sprintf("%v.%v.svc", alertmanagerService.Name, cr.Namespace),
				},
				BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
			},
		},
	}
}

func (r *Reconciler) reconcilePrometheus(ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex) error {
	secrets, err := r.getTokenSecrets(ctx, cr)
	if err != nil {
		return err
	}

	proxySecret := model.GetPrometheusProxySecret(cr)
	sa := model.GetPrometheusServiceAccount(cr)

	route := model.GetPrometheusRoute(cr)
	selector := client.ObjectKey{
		Namespace: route.Namespace,
		Name:      route.Name,
	}

	err = r.client.Get(ctx, selector, route)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	host := ""
	if utils.IsRouteReady(route) {
		host = route.Spec.Host
	}

	secrets = append(secrets, proxySecret.Name)
	secrets = append(secrets, "prometheus-k8s-tls")

	var remoteWrites []prometheusv1.RemoteWriteSpec
	for _, index := range indexes {
		patterns, err := r.getRemoteWriteIndex(index)
		if err != nil {
			return err
		}

		remoteWrite, err := r.getRemoteWriteSpec(index, secrets, patterns)
		if err != nil {
			return err
		}
		remoteWrites = append(remoteWrites, *remoteWrite)
	}

	prometheus := model.GetPrometheus(cr)
	_, err = controllerutil.CreateOrUpdate(ctx, r.client, prometheus, func() error {
		prometheus.Spec = prometheusv1.PrometheusSpec{
			ServiceAccountName: sa.Name,
			ExternalURL:        fmt.Sprintf("https://%v", host),
			AdditionalScrapeConfigs: &kv1.SecretKeySelector{
				LocalObjectReference: kv1.LocalObjectReference{
					Name: "additional-scrape-configs",
				},
				Key: "additional-scrape-config.yaml",
			},
			ExternalLabels: map[string]string{
				"cluster_id": cr.Status.ClusterID,
			},
			PodMonitorSelector: &v12.LabelSelector{
				MatchLabels: model.GetResourceLabels(),
			},
			ServiceMonitorSelector: &v12.LabelSelector{
				MatchLabels: model.GetResourceLabels(),
			},
			RuleSelector: &v12.LabelSelector{
				MatchLabels: model.GetResourceLabels(),
			},
			RemoteWrite: remoteWrites,
			Alerting:    r.getAlerting(cr),
			Secrets:     secrets,
			Containers: []kv1.Container{
				{
					Name:  "oauth-proxy",
					Image: "quay.io/openshift/origin-oauth-proxy:4.2",
					Args: []string{
						"-provider=openshift",
						"-https-address=:9091",
						"-http-address=",
						"-email-domain=*",
						"-upstream=http://localhost:9090",
						fmt.Sprintf("-openshift-service-account=%v", sa.Name),
						"-openshift-sar={\"resource\": \"namespaces\", \"verb\": \"get\"}",
						"-openshift-delegate-urls={\"/\": {\"resource\": \"namespaces\", \"verb\": \"get\"}}",
						"-tls-cert=/etc/tls/private/tls.crt",
						"-tls-key=/etc/tls/private/tls.key",
						"-client-secret-file=/var/run/secrets/kubernetes.io/serviceaccount/token",
						"-cookie-secret-file=/etc/proxy/secrets/session_secret",
						"-openshift-ca=/etc/pki/tls/cert.pem",
						"-openshift-ca=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
						"-skip-auth-regex=^/metrics",
					},
					Env: []kv1.EnvVar{
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
					Ports: []kv1.ContainerPort{
						{
							Name:          "proxy",
							ContainerPort: 9091,
						},
					},
					VolumeMounts: []kv1.VolumeMount{
						{
							Name:      "secret-prometheus-k8s-tls",
							MountPath: "/etc/tls/private",
						},
						{
							Name:      fmt.Sprintf("secret-%v", proxySecret.Name),
							MountPath: "/etc/proxy/secrets",
						},
					},
				},
			},
		}
		if cr.Spec.Storage != nil && cr.Spec.Storage.PrometheusStorageSpec != nil {
			prometheus.Spec.Storage = cr.Spec.Storage.PrometheusStorageSpec
		}
		if cr.Spec.Tolerations != nil {
			prometheus.Spec.Tolerations = cr.Spec.Tolerations
		}
		if cr.Spec.Affinity != nil {
			prometheus.Spec.Affinity = cr.Spec.Affinity
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
