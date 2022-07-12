package model

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	t "text/template"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"html/template"

	routev1 "github.com/openshift/api/route/v1"
	coreosv1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	v13 "k8s.io/api/core/v1"
	v14 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var defaultPrometheusLabelSelectors = map[string]string{"app": "strimzi"}

const (
	PrometheusVersion        = "v2.22.2"
	PrometheusDefaultStorage = "250Gi"
	PrometheusOldRouteName   = "kafka-prometheus"
)

func GetPrometheusNamespace(cr *v1.Observability) *v13.Namespace {
	return &v13.Namespace{
		ObjectMeta: v12.ObjectMeta{
			Name: cr.GetPrometheusOperatorNamespace(),
		},
	}
}

func GetDefaultNamePrometheus(cr *v1.Observability) string {
	if cr.Spec.SelfContained != nil && cr.Spec.PrometheusDefaultName != "" {
		return cr.Spec.PrometheusDefaultName
	}
	return "observability-prometheus"
}

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
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
	}
}

func GetPrometheusSubscription(cr *v1.Observability) *v1alpha1.Subscription {
	return &v1alpha1.Subscription{
		ObjectMeta: v12.ObjectMeta{
			Name:      "prometheus-subscription",
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
	}
}

func GetPrometheusCatalogSource(cr *v1.Observability) *v1alpha1.CatalogSource {
	return &v1alpha1.CatalogSource{
		ObjectMeta: v12.ObjectMeta{
			Name:      "prometheus-catalogsource",
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
	}
}

func GetPrometheusProxySecret(cr *v1.Observability) *v13.Secret {
	return &v13.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      "prometheus-proxy",
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
	}
}

func GetPrometheusTLSSecret(cr *v1.Observability) *v13.Secret {
	return &v13.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      "prometheus-k8s-tls",
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
	}
}

func GetPrometheusServiceAccount(cr *v1.Observability) *v13.ServiceAccount {
	route := GetPrometheusRoute(cr)
	redirect := fmt.Sprintf("{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"%s\"}}", route.Name)

	return &v13.ServiceAccount{
		ObjectMeta: v12.ObjectMeta{
			Name:      GetDefaultNamePrometheus(cr),
			Namespace: cr.GetPrometheusOperatorNamespace(),
			Annotations: map[string]string{
				"serviceaccounts.openshift.io/oauth-redirectreference.primary": redirect,
			},
		},
	}
}

func GetPrometheusService(cr *v1.Observability) *v13.Service {
	return &v13.Service{
		ObjectMeta: v12.ObjectMeta{
			Name:      GetDefaultNamePrometheus(cr),
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
	}
}

func GetPrometheusClusterRole(cr *v1.Observability) *v14.ClusterRole {
	return &v14.ClusterRole{
		ObjectMeta: v12.ObjectMeta{
			Name: GetDefaultNamePrometheus(cr),
		},
	}
}

func GetPrometheusClusterRoleBinding(cr *v1.Observability) *v14.ClusterRoleBinding {
	return &v14.ClusterRoleBinding{
		ObjectMeta: v12.ObjectMeta{
			Name: GetDefaultNamePrometheus(cr),
		},
	}
}

func GetPrometheusRoute(cr *v1.Observability) *routev1.Route {
	return &routev1.Route{
		ObjectMeta: v12.ObjectMeta{
			Name:      GetDefaultNamePrometheus(cr),
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
	}
}

func GetFederationConfigBearerToken(patterns []string) ([]byte, error) {
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
  bearer_token_file: "/var/run/secrets/kubernetes.io/serviceaccount/token"
  tls_config:
    insecure_skip_verify: true
`

	template := t.Must(t.New("template").Parse(config))
	var buffer bytes.Buffer
	err := template.Execute(&buffer, struct {
		Patterns string
	}{
		Patterns: strings.Join(patterns, ","),
	})

	return buffer.Bytes(), err
}

func GetPrometheusAdditionalScrapeConfig(cr *v1.Observability) *v13.Secret {
	return &v13.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      "additional-scrape-configs",
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
	}
}

func GetPrometheusBlackBoxConfig(cr *v1.Observability) *v13.ConfigMap {
	return &v13.ConfigMap{
		ObjectMeta: v12.ObjectMeta{
			Name:      "black-box-config",
			Namespace: cr.GetPrometheusOperatorNamespace(),
			Labels: map[string]string{
				"managed-by": "observability-operator",
			},
		},
	}
}

func GetDefaultBlackBoxConfig(cr *v1.Observability, ctx context.Context, client k8sclient.Client) ([]byte, string, error) {
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
        key_file: /etc/tls/private/tls.key{{ end }}{{ if .HasBlackboxBearerToken  }}
      bearer_token: {{ .BearerToken }}{{ end }}
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

	// Get bearer token if it exists
	hasBlackboxBearerToken, token := GetBlackboxBearerToken(cr, ctx, client)

	var buffer bytes.Buffer
	params := struct {
		SelfSignedCerts        bool
		HasBlackboxBearerToken bool
		BearerToken            string
	}{
		SelfSignedCerts:        cr.SelfSignedCerts(),
		HasBlackboxBearerToken: hasBlackboxBearerToken,
		BearerToken:            token,
	}

	err = parsed.Execute(&buffer, &params)
	if err != nil {
		return nil, "", err
	}

	hash := sha256.Sum256(buffer.Bytes())
	return buffer.Bytes(), fmt.Sprintf("%x", hash), nil
}

func GetBlackboxBearerToken(cr *v1.Observability, ctx context.Context, client k8sclient.Client) (bool, string) {
	hasSecret, secretName := cr.HasBlackboxBearerTokenSecret()
	if hasSecret {
		secret := &v13.Secret{}
		selector := k8sclient.ObjectKey{
			Namespace: cr.Namespace,
			Name:      secretName,
		}

		err := client.Get(ctx, selector, secret)
		if err != nil {
			return false, ""
		}

		return true, string(secret.Data["token"])

	}
	return false, ""
}

func GetPrometheus(cr *v1.Observability) *prometheusv1.Prometheus {
	return &prometheusv1.Prometheus{
		ObjectMeta: v12.ObjectMeta{
			Name:      GetDefaultNamePrometheus(cr),
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
	}
}

func GetDeadmansSwitch(cr *v1.Observability) *prometheusv1.PrometheusRule {
	return &prometheusv1.PrometheusRule{
		ObjectMeta: v12.ObjectMeta{
			Name:      "generated-deadmansswitch",
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
	}
}

// Label Selectors

func GetPrometheusPodMonitorLabelSelectors(cr *v1.Observability, indexes []v1.RepositoryIndex) *v12.LabelSelector {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.PodMonitorLabelSelector != nil {
		return cr.Spec.SelfContained.PodMonitorLabelSelector
	}

	if cr.OverrideSelectors() && cr.Spec.SelfContained.PodMonitorLabelSelector == nil {
		return &v12.LabelSelector{}
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

	if cr.OverrideSelectors() && cr.Spec.SelfContained.ServiceMonitorLabelSelector == nil {
		return &v12.LabelSelector{}
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

	if cr.OverrideSelectors() && cr.Spec.SelfContained.RuleLabelSelector == nil {
		return &v12.LabelSelector{}
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

	if cr.OverrideSelectors() && cr.Spec.SelfContained.ProbeLabelSelector == nil {
		return &v12.LabelSelector{}
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

	if cr.OverrideSelectors() && cr.Spec.SelfContained.PodMonitorNamespaceSelector == nil {
		return &v12.LabelSelector{}
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

	if cr.OverrideSelectors() && cr.Spec.SelfContained.ServiceMonitorNamespaceSelector == nil {
		return &v12.LabelSelector{}
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

	if cr.OverrideSelectors() && cr.Spec.SelfContained.RuleNamespaceSelector == nil {
		return &v12.LabelSelector{}
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

	if cr.OverrideSelectors() && cr.Spec.SelfContained.ProbeNamespaceSelector == nil {
		return &v12.LabelSelector{}
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

func GetPrometheusVersion(cr *v1.Observability) string {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.PrometheusVersion != "" {
		return cr.Spec.SelfContained.PrometheusVersion
	}
	return PrometheusVersion
}

func GetPrometheusResourceRequirement(cr *v1.Observability) *v13.ResourceRequirements {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.PrometheusResourceRequirement != nil {
		return cr.Spec.SelfContained.PrometheusResourceRequirement
	}
	return &v13.ResourceRequirements{}
}

func GetPrometheusOperatorResourceRequirement(cr *v1.Observability) *v13.ResourceRequirements {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.PrometheusOperatorResourceRequirement != nil {
		return cr.Spec.SelfContained.PrometheusOperatorResourceRequirement
	}
	return &v13.ResourceRequirements{}
}
func GetPrometheusStorageSize(cr *v1.Observability, indexes []v1.RepositoryIndex) string {
	customPrometheusStorageSize := PrometheusDefaultStorage
	if cr.Spec.SelfContained != nil &&
		cr.Spec.Storage != nil &&
		cr.Spec.Storage.PrometheusStorageSpec != nil &&
		cr.Spec.Storage.PrometheusStorageSpec.VolumeClaimTemplate.Spec.Resources.Requests != nil &&
		cr.Spec.Storage.PrometheusStorageSpec.VolumeClaimTemplate.Spec.Resources.Requests.Storage() != nil {
		customPrometheusStorageSize = cr.Spec.Storage.PrometheusStorageSpec.VolumeClaimTemplate.Spec.Resources.Requests.Storage().String()
	}
	prometheusConfig := getPrometheusRepositoryIndexConfig(indexes)
	if prometheusConfig != nil && prometheusConfig.OverridePrometheusPvcSize != "" {
		customPrometheusStorageSize = prometheusConfig.OverridePrometheusPvcSize
	}
	return customPrometheusStorageSize
}
