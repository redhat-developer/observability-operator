package model

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	t "text/template"

	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	routev1 "github.com/openshift/api/route/v1"
	coreosv1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"html/template"
	v13 "k8s.io/api/core/v1"
	v14 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var defaultPrometheusLabelSelectors = map[string]string{"app": "strimzi"}

func GetPrometheusAuthTokenLifetimes(cr *v1.Observability) *v13.ConfigMap {
	return &v13.ConfigMap{
		ObjectMeta: v12.ObjectMeta{
			Name:      "observatorium-token-lifetimes",
			Namespace: cr.Namespace,
		},
	}
}

func GetPrometheusOperatorgroup(cr *v1.Observability) *coreosv1.OperatorGroup {
	return &coreosv1.OperatorGroup{
		ObjectMeta: v12.ObjectMeta{
			Name:      "observability-operatorgroup",
			Namespace: cr.Namespace,
		},
	}
}

func GetPrometheusSubscription(cr *v1.Observability) *v1alpha1.Subscription {
	return &v1alpha1.Subscription{
		ObjectMeta: v12.ObjectMeta{
			Name:      "prometheus-subscription",
			Namespace: cr.Namespace,
		},
	}
}

func GetPrometheusCatalogSource(cr *v1.Observability) *v1alpha1.CatalogSource {
	return &v1alpha1.CatalogSource{
		ObjectMeta: v12.ObjectMeta{
			Name:      "prometheus-catalogsource",
			Namespace: cr.Namespace,
		},
	}
}

func GetPrometheusProxySecret(cr *v1.Observability) *v13.Secret {
	return &v13.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      "prometheus-proxy",
			Namespace: cr.Namespace,
		},
	}
}

func GetPrometheusTLSSecret(cr *v1.Observability) *v13.Secret {
	return &v13.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      "prometheus-k8s-tls",
			Namespace: cr.Namespace,
		},
	}
}

func GetPrometheusServiceAccount(cr *v1.Observability) *v13.ServiceAccount {
	route := GetPrometheusRoute(cr)
	redirect := fmt.Sprintf("{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"%s\"}}", route.Name)

	return &v13.ServiceAccount{
		ObjectMeta: v12.ObjectMeta{
			Name:      "kafka-prometheus",
			Namespace: cr.Namespace,
			Annotations: map[string]string{
				"serviceaccounts.openshift.io/oauth-redirectreference.primary": redirect,
			},
		},
	}
}

func GetPrometheusService(cr *v1.Observability) *v13.Service {
	return &v13.Service{
		ObjectMeta: v12.ObjectMeta{
			Name:      "kafka-prometheus",
			Namespace: cr.Namespace,
		},
	}
}

func GetPrometheusClusterRole() *v14.ClusterRole {
	return &v14.ClusterRole{
		ObjectMeta: v12.ObjectMeta{
			Name: "kafka-prometheus",
		},
	}
}

func GetPrometheusClusterRoleBinding() *v14.ClusterRoleBinding {
	return &v14.ClusterRoleBinding{
		ObjectMeta: v12.ObjectMeta{
			Name: "kafka-prometheus",
		},
	}
}

func GetPrometheusRoute(cr *v1.Observability) *routev1.Route {
	return &routev1.Route{
		ObjectMeta: v12.ObjectMeta{
			Name:      "kafka-prometheus",
			Namespace: cr.Namespace,
		},
	}
}

func GetFederationConfig(user, pass string, patterns []string) ([]byte, error) {
	const config = `
- job_name: openshift-monitoring-federation
  honor_labels: true
  kubernetes_sd_configs:
    - role: service
      namespaces:
        names:
          - openshift-monitoring
  scrape_interval: 120s
  scrape_timeout: 60s
  metrics_path: /federate
  relabel_configs:
    - action: keep
      source_labels: [ '__meta_kubernetes_service_name' ]
      regex: prometheus-k8s
    - action: keep
      source_labels: [ '__meta_kubernetes_service_port_name' ]
      regex: web
  params:
    match[]: [{{ .Patterns }}]
  scheme: https
  tls_config:
    insecure_skip_verify: true
  basic_auth:
    username: {{ .User }}
    password: {{ .Pass }}
`

	template := t.Must(t.New("template").Parse(config))
	var buffer bytes.Buffer
	err := template.Execute(&buffer, struct {
		User     string
		Pass     string
		Patterns string
	}{
		User:     user,
		Pass:     pass,
		Patterns: strings.Join(patterns, ","),
	})

	return buffer.Bytes(), err
}

func GetPrometheusAdditionalScrapeConfig(cr *v1.Observability) *v13.Secret {
	return &v13.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      "additional-scrape-configs",
			Namespace: cr.Namespace,
		},
	}
}

func GetPrometheusBlackBoxConfig(cr *v1.Observability) *v13.ConfigMap {
	return &v13.ConfigMap{
		ObjectMeta: v12.ObjectMeta{
			Name:      "black-box-config",
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"managed-by": "observability-operator",
			},
		},
	}
}

func GetDefaultBlackBoxConfig(cr *v1.Observability) ([]byte, string, error) {
	blackBoxConfig := `modules:
  http_extern_2xx:
    prober: http
    http:
      preferred_ip_protocol: ip4
  http_2xx:
    prober: http
    http:
      preferred_ip_protocol: ip4{{ if .SelfSignedCerts }}
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        cert_file: /etc/tls/private/tls.crt
        key_file: /etc/tls/private/tls.key{{ end }}
  http_post_2xx:
    prober: http
    http:
      method: POST
      preferred_ip_protocol: ip4{{ if .SelfSignedCerts }}
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        cert_file: /etc/tls/private/tls.crt
        key_file: /etc/tls/private/tls.key{{ end }}`

	parser := template.New("blackbox-config")
	parsed, err := parser.Parse(blackBoxConfig)
	if err != nil {
		return nil, "", err
	}

	var buffer bytes.Buffer
	params := struct {
		SelfSignedCerts bool
	}{
		SelfSignedCerts: cr.SelfSignedCerts(),
	}

	err = parsed.Execute(&buffer, &params)
	if err != nil {
		return nil, "", err
	}

	hash := sha256.Sum256(buffer.Bytes())
	return buffer.Bytes(), fmt.Sprintf("%x", hash), nil
}

func GetPrometheus(cr *v1.Observability) *prometheusv1.Prometheus {
	return &prometheusv1.Prometheus{
		ObjectMeta: v12.ObjectMeta{
			Name:      "kafka-prometheus",
			Namespace: cr.Namespace,
		},
	}
}

// Label Selectors

func GetPrometheusPodMonitorLabelSelectors(cr *v1.Observability, indexes []v1.RepositoryIndex) *v12.LabelSelector {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.PodMonitorLabelSelector != nil {
		return cr.Spec.SelfContained.PodMonitorLabelSelector
	}

	prometheusConfig := getPrometheusRepositoryIndexConfig(indexes)
	if prometheusConfig != nil && prometheusConfig.PodMonitorLabelSelector != nil {
		return prometheusConfig.PodMonitorLabelSelector
	}

	return &v12.LabelSelector{
		MatchLabels: defaultPrometheusLabelSelectors,
	}
}

func GetPrometheusServiceMonitorLabelSelectors(cr *v1.Observability, indexes []v1.RepositoryIndex) *v12.LabelSelector {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.ServiceMonitorLabelSelector != nil {
		return cr.Spec.SelfContained.ServiceMonitorLabelSelector
	}

	prometheusConfig := getPrometheusRepositoryIndexConfig(indexes)
	if prometheusConfig != nil && prometheusConfig.ServiceMonitorLabelSelector != nil {
		return prometheusConfig.ServiceMonitorLabelSelector
	}

	return &v12.LabelSelector{
		MatchLabels: defaultPrometheusLabelSelectors,
	}
}

func GetPrometheusRuleLabelSelectors(cr *v1.Observability, indexes []v1.RepositoryIndex) *v12.LabelSelector {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.RuleLabelSelector != nil {
		return cr.Spec.SelfContained.RuleLabelSelector
	}

	prometheusConfig := getPrometheusRepositoryIndexConfig(indexes)
	if prometheusConfig != nil && prometheusConfig.RuleLabelSelector != nil {
		return prometheusConfig.RuleLabelSelector
	}

	return &v12.LabelSelector{
		MatchLabels: defaultPrometheusLabelSelectors,
	}
}

func GetProbeLabelSelectors(cr *v1.Observability, indexes []v1.RepositoryIndex) *v12.LabelSelector {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.ProbeLabelSelector != nil {
		return cr.Spec.SelfContained.ProbeLabelSelector
	}

	prometheusConfig := getPrometheusRepositoryIndexConfig(indexes)
	if prometheusConfig != nil && prometheusConfig.ProbeLabelSelector != nil {
		return prometheusConfig.ProbeLabelSelector
	}

	return &v12.LabelSelector{
		MatchLabels: defaultPrometheusLabelSelectors,
	}
}

// Namespace selectors

func GetPrometheusPodMonitorNamespaceSelectors(cr *v1.Observability, indexes []v1.RepositoryIndex) *v12.LabelSelector {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.PodMonitorNamespaceSelector != nil {
		return cr.Spec.SelfContained.PodMonitorNamespaceSelector
	}
	prometheusConfig := getPrometheusRepositoryIndexConfig(indexes)
	if prometheusConfig != nil && prometheusConfig.PodMonitorNamespaceSelector != nil {
		return prometheusConfig.PodMonitorNamespaceSelector
	}
	return nil
}

func GetPrometheusServiceMonitorNamespaceSelectors(cr *v1.Observability, indexes []v1.RepositoryIndex) *v12.LabelSelector {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.ServiceMonitorNamespaceSelector != nil {
		return cr.Spec.SelfContained.ServiceMonitorNamespaceSelector
	}
	prometheusConfig := getPrometheusRepositoryIndexConfig(indexes)
	if prometheusConfig != nil && prometheusConfig.ServiceMonitorNamespaceSelector != nil {
		return prometheusConfig.ServiceMonitorNamespaceSelector
	}
	return nil
}

func GetPrometheusRuleNamespaceSelectors(cr *v1.Observability, indexes []v1.RepositoryIndex) *v12.LabelSelector {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.RuleNamespaceSelector != nil {
		return cr.Spec.SelfContained.RuleNamespaceSelector
	}
	prometheusConfig := getPrometheusRepositoryIndexConfig(indexes)
	if prometheusConfig != nil && prometheusConfig.RuleNamespaceSelector != nil {
		return prometheusConfig.RuleNamespaceSelector
	}
	return nil
}

func GetProbeNamespaceSelectors(cr *v1.Observability, indexes []v1.RepositoryIndex) *v12.LabelSelector {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.ProbeNamespaceSelector != nil {
		return cr.Spec.SelfContained.ProbeNamespaceSelector
	}
	prometheusConfig := getPrometheusRepositoryIndexConfig(indexes)
	if prometheusConfig != nil && prometheusConfig.ProbeNamespaceSelector != nil {
		return prometheusConfig.ProbeNamespaceSelector
	}
	return nil
}

// returns the Prometheus configuration from the repository index
func getPrometheusRepositoryIndexConfig(indexes []v1.RepositoryIndex) *v1.PrometheusIndex {
	if len(indexes) > 0 {
		// We should only have one Prometheus CR for the whole cluster. However, we cannot merge
		// all of the label selectors from all of the repository index config as this will result
		// in an AND requirement. Since we do not use multiple repositories on the same cluster just yet,
		// there should only be one index available in the repository index list.
		// This needs to be changed once we start using multiple repository configurations on the same cluster.
		if indexes[0].Config != nil {
			return indexes[0].Config.Prometheus
		}
	}
	return &v1.PrometheusIndex{}
}
