package model

import (
	"testing"

	. "github.com/onsi/gomega"
	routev1 "github.com/openshift/api/route/v1"
	coreosv1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	corev1 "k8s.io/api/core/v1"
	v14 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	objectMetaWithPrometheusName       = v12.ObjectMeta{Name: defaultPrometheusName}
	testNamespace                      = "testNamespace"
	defaultPrometheusName              = "kafka-prometheus"
	serviceAccountPrometheusAnnotation = map[string]string{"serviceaccounts.openshift.io/oauth-redirectreference.primary": "{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"kafka-prometheus\"}}"}
	testPattern                        = []string{"test1", "test2"}
	testFederationConfig               = `
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
    match[]: [test1,test2]
  scheme: https
  tls_config:
    insecure_skip_verify: true
  basic_auth:
    username: testuser
    password: testpass
`
	configAsByteArray = []byte(testFederationConfig)
	testRepoIndexes   = []v1.RepositoryIndex{
		{
			Config: &v1.RepositoryConfig{
				Grafana: &v1.GrafanaIndex{
					DashboardLabelSelector: labelSelectorWithNamespace,
				},
				Prometheus: &v1.PrometheusIndex{
					ProbeNamespaceSelector:          labelSelectorWithNamespace,
					PodMonitorLabelSelector:         labelSelectorWithNamespace,
					ServiceMonitorLabelSelector:     labelSelectorWithNamespace,
					RuleLabelSelector:               labelSelectorWithNamespace,
					ProbeLabelSelector:              labelSelectorWithNamespace,
					PodMonitorNamespaceSelector:     labelSelectorWithNamespace,
					ServiceMonitorNamespaceSelector: labelSelectorWithNamespace,
					RuleNamespaceSelector:           labelSelectorWithNamespace,
					OverridePrometheusPvcSize:       "test-quantity",
				},
				Alertmanager: &v1.AlertmanagerIndex{
					OverrideAlertmanagerPvcSize: "test-quantity",
				},
			},
		},
		{
			Config: &v1.RepositoryConfig{
				Promtail: &v1.PromtailIndex{
					DaemonSetLabelSelector: &v12.LabelSelector{
						MatchLabels: map[string]string{
							"app": "promtail-test",
						},
					},
				},
			},
		},
	}
)

func TestPrometheusResources_GetDefaultNamePrometheus(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "returns 'kafka-prometheus' if NOT self contained",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: defaultPrometheusName,
		},
		{
			name: "returns CR default Prometheus name if self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
					obsCR.Spec.PrometheusDefaultName = "test"
				}),
			},
			want: "test",
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDefaultNamePrometheus(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusAuthTokenLifetimes(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *corev1.ConfigMap
	}{
		{
			name: "returns 'observatorium-token-lifetimes' ConfigMap",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      "observatorium-token-lifetimes",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusAuthTokenLifetimes(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusOperatorgroup(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *coreosv1.OperatorGroup
	}{
		{
			name: "returns 'observability-operatorgroup' OperatorGroup",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &coreosv1.OperatorGroup{
				ObjectMeta: v12.ObjectMeta{
					Name:      "observability-operatorgroup",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusOperatorgroup(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusSubscription(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *v1alpha1.Subscription
	}{
		{
			name: "returns 'prometheus-subscription' Subscription",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &v1alpha1.Subscription{
				ObjectMeta: v12.ObjectMeta{
					Name:      "prometheus-subscription",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusSubscription(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusCatalogSource(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *v1alpha1.CatalogSource
	}{
		{
			name: "returns 'prometheus-catalogsource' CatalogSource",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &v1alpha1.CatalogSource{
				ObjectMeta: v12.ObjectMeta{
					Name:      "prometheus-catalogsource",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusCatalogSource(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusProxySecret(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *corev1.Secret
	}{
		{
			name: "returns 'prometheus-proxy' Secret",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "prometheus-proxy",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusProxySecret(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusTLSSecret(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *corev1.Secret
	}{
		{
			name: "returns 'prometheus-k8s-tls' Secret",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "prometheus-k8s-tls",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusTLSSecret(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusServiceAccount(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *corev1.ServiceAccount
	}{
		{
			name: "returns prometheus service account",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.ServiceAccount{
				ObjectMeta: v12.ObjectMeta{
					Name:        defaultPrometheusName,
					Namespace:   testNamespace,
					Annotations: serviceAccountPrometheusAnnotation,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusServiceAccount(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusService(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *corev1.Service
	}{
		{
			name: "returns prometheus service",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.Service{
				ObjectMeta: v12.ObjectMeta{
					Name:      defaultPrometheusName,
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusService(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusClusterRole(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *v14.ClusterRole
	}{
		{
			name: "returns Prometheus ClusterRole",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &v14.ClusterRole{
				ObjectMeta: objectMetaWithPrometheusName,
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusClusterRole(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusClusterRoleBinding(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *v14.ClusterRoleBinding
	}{
		{
			name: "returns Prometheus ClusterRoleBinding",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &v14.ClusterRoleBinding{
				ObjectMeta: objectMetaWithPrometheusName,
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusClusterRoleBinding(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusRoute(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *routev1.Route
	}{
		{
			name: "returns Prometheus route",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.ObjectMeta = objectMetaWithNamespace
				}),
			},
			want: &routev1.Route{
				ObjectMeta: v12.ObjectMeta{
					Name:      defaultPrometheusName,
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusRoute(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetFederationConfig(t *testing.T) {
	type args struct {
		user     string
		pass     string
		patterns []string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []byte
	}{
		{
			name: "returns correct federation config with no error",
			args: args{
				user:     "testuser",
				pass:     "testpass",
				patterns: testPattern,
			},
			wantErr: false,
			want:    configAsByteArray,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetFederationConfig(tt.args.user, tt.args.pass, tt.args.patterns)
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusAdditionalScrapeConfig(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *corev1.Secret
	}{
		{
			name: "returns Prometheus additional scrape configs",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.ObjectMeta = objectMetaWithNamespace
				}),
			},
			want: &corev1.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "additional-scrape-configs",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusAdditionalScrapeConfig(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusBlackBoxConfig(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *corev1.ConfigMap
	}{
		{
			name: "returns Prometheus black box config",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.ObjectMeta = objectMetaWithNamespace
				}),
			},
			want: &corev1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      "black-box-config",
					Namespace: testNamespace,
					Labels: map[string]string{
						"managed-by": "observability-operator",
					},
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusBlackBoxConfig(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheus(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *monitoringv1.Prometheus
	}{
		{
			name: "returns Prometheus",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.ObjectMeta = objectMetaWithNamespace
				}),
			},
			want: &monitoringv1.Prometheus{
				ObjectMeta: v12.ObjectMeta{
					Name:      defaultPrometheusName,
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheus(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetDeadmansSwitch(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *monitoringv1.PrometheusRule
	}{
		{
			name: "returns PrometheusRule for Dead Mans Switch alert",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.ObjectMeta = objectMetaWithNamespace
				}),
			},
			want: &monitoringv1.PrometheusRule{
				ObjectMeta: v12.ObjectMeta{
					Name:      "generated-deadmansswitch",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDeadmansSwitch(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusPodMonitorLabelSelectors(t *testing.T) {
	type args struct {
		cr      *v1.Observability
		indexes []v1.RepositoryIndex
	}

	tests := []struct {
		name string
		args args
		want *v12.LabelSelector
	}{
		{
			name: "returns CR PodMonitorLabelSelector when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						PodMonitorLabelSelector: labelSelectorWithNamespace,
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns blank LabelSelector when self contained selector is nil and selectors are overridden",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						PodMonitorLabelSelector: nil,
						OverrideSelectors:       &([]bool{true})[0],
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: &v12.LabelSelector{},
		},
		{
			name: "returns pod monitor LabelSelector from repo Prometheus Index",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: testRepoIndexes,
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns default pod monitor LabelSelector when self contained is nil and no repo config",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: []v1.RepositoryIndex{},
			},
			want: &v12.LabelSelector{
				MatchLabels: defaultPrometheusLabelSelectors,
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusPodMonitorLabelSelectors(tt.args.cr, tt.args.indexes)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusServiceMonitorLabelSelectors(t *testing.T) {
	type args struct {
		cr      *v1.Observability
		indexes []v1.RepositoryIndex
	}

	tests := []struct {
		name string
		args args
		want *v12.LabelSelector
	}{
		{
			name: "returns CR ServiceMonitorLabelSelector when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						ServiceMonitorLabelSelector: labelSelectorWithNamespace,
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns blank LabelSelector when self contained selector is nil and selectors are overridden",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						ServiceMonitorLabelSelector: nil,
						OverrideSelectors:           &([]bool{true})[0],
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: &v12.LabelSelector{},
		},
		{
			name: "returns service monitor LabelSelector from repo Prometheus Index",
			args: args{
				cr: &v1.Observability{
					Spec: v1.ObservabilitySpec{
						SelfContained: nil,
					},
				},
				indexes: testRepoIndexes,
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns default service monitor LabelSelector when self contained is nil and no repo config",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: []v1.RepositoryIndex{},
			},
			want: &v12.LabelSelector{
				MatchLabels: defaultPrometheusLabelSelectors,
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusServiceMonitorLabelSelectors(tt.args.cr, tt.args.indexes)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusRuleLabelSelectors(t *testing.T) {
	type args struct {
		cr      *v1.Observability
		indexes []v1.RepositoryIndex
	}

	tests := []struct {
		name string
		args args
		want *v12.LabelSelector
	}{
		{
			name: "returns CR RuleLabelSelector when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						RuleLabelSelector: labelSelectorWithNamespace,
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns blank LabelSelector when self contained selector is nil and selectors are overridden",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						RuleLabelSelector: nil,
						OverrideSelectors: &([]bool{true})[0],
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: &v12.LabelSelector{},
		},
		{
			name: "returns Prometheus rule LabelSelector from repo Prometheus Index",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: testRepoIndexes,
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns default Prometheus rule LabelSelector when self contained is nil and no repo config",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: []v1.RepositoryIndex{},
			},
			want: &v12.LabelSelector{
				MatchLabels: defaultPrometheusLabelSelectors,
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusRuleLabelSelectors(tt.args.cr, tt.args.indexes)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetProbeLabelSelectors(t *testing.T) {
	type args struct {
		cr      *v1.Observability
		indexes []v1.RepositoryIndex
	}

	tests := []struct {
		name string
		args args
		want *v12.LabelSelector
	}{
		{
			name: "returns CR ProbeLabelSelector when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						ProbeLabelSelector: labelSelectorWithNamespace,
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns blank LabelSelector when self contained selector is nil and selectors are overridden",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						RuleLabelSelector: nil,
						OverrideSelectors: &([]bool{true})[0],
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: &v12.LabelSelector{},
		},
		{
			name: "returns probe LabelSelector from repo Prometheus Index",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: testRepoIndexes,
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns default probe LabelSelector when self contained is nil and no repo config",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: []v1.RepositoryIndex{},
			},
			want: &v12.LabelSelector{
				MatchLabels: defaultPrometheusLabelSelectors,
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProbeLabelSelectors(tt.args.cr, tt.args.indexes)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusPodMonitorNamespaceSelectors(t *testing.T) {
	type args struct {
		cr      *v1.Observability
		indexes []v1.RepositoryIndex
	}

	tests := []struct {
		name string
		args args
		want *v12.LabelSelector
	}{
		{
			name: "returns CR PodMonitorNamespaceSelector when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						PodMonitorNamespaceSelector: labelSelectorWithNamespace,
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns blank NamespaceSelector when self contained selector is nil and selectors are overridden",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						PodMonitorNamespaceSelector: nil,
						OverrideSelectors:           &([]bool{true})[0],
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: &v12.LabelSelector{},
		},
		{
			name: "returns pod monitor NamespaceSelector from repo Prometheus Index",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: testRepoIndexes,
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns nil when self contained is nil and no repo config",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: []v1.RepositoryIndex{},
			},
			want: nil,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusPodMonitorNamespaceSelectors(tt.args.cr, tt.args.indexes)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusServiceMonitorNamespaceSelectors(t *testing.T) {
	type args struct {
		cr      *v1.Observability
		indexes []v1.RepositoryIndex
	}

	tests := []struct {
		name string
		args args
		want *v12.LabelSelector
	}{
		{
			name: "returns CR ServiceMonitorNamespaceSelector when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						ServiceMonitorNamespaceSelector: labelSelectorWithNamespace,
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns blank LabelSelector when self contained selector is nil and selectors are overridden",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						ServiceMonitorNamespaceSelector: nil,
						OverrideSelectors:               &([]bool{true})[0],
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: &v12.LabelSelector{},
		},
		{
			name: "returns service monitor NamespaceSelector from repo Prometheus Index",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: testRepoIndexes,
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns nil when self contained is nil and no repo config",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: []v1.RepositoryIndex{},
			},
			want: nil,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusServiceMonitorNamespaceSelectors(tt.args.cr, tt.args.indexes)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusRuleNamespaceSelectors(t *testing.T) {
	type args struct {
		cr      *v1.Observability
		indexes []v1.RepositoryIndex
	}

	tests := []struct {
		name string
		args args
		want *v12.LabelSelector
	}{
		{
			name: "returns CR RuleNamespaceSelector when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						RuleNamespaceSelector: labelSelectorWithNamespace,
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns blank NamespaceSelector when self contained selector is nil and selectors are overridden",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						RuleNamespaceSelector: nil,
						OverrideSelectors:     &([]bool{true})[0],
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: &v12.LabelSelector{},
		},
		{
			name: "returns Prometheus rule NamespaceSelector from repo Prometheus Index",
			args: args{
				cr: &v1.Observability{
					Spec: v1.ObservabilitySpec{
						SelfContained: nil,
					},
				},
				indexes: testRepoIndexes,
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns nil when self contained is nil and no repo config",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: []v1.RepositoryIndex{},
			},
			want: nil,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusRuleNamespaceSelectors(tt.args.cr, tt.args.indexes)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetProbeNamespaceSelectors(t *testing.T) {
	type args struct {
		cr      *v1.Observability
		indexes []v1.RepositoryIndex
	}

	tests := []struct {
		name string
		args args
		want *v12.LabelSelector
	}{
		{
			name: "returns CR ProbeNamespaceSelector when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						ProbeNamespaceSelector: labelSelectorWithNamespace,
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns blank NamespaceSelector when self contained selector is nil and selectors are overridden",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						ProbeNamespaceSelector: nil,
						OverrideSelectors:      &([]bool{true})[0],
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: &v12.LabelSelector{},
		},
		{
			name: "returns probe NamespaceSelector from repo Prometheus Index",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: testRepoIndexes,
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns nil when self contained is nil and no repo config",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: []v1.RepositoryIndex{},
			},
			want: nil,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProbeNamespaceSelectors(tt.args.cr, tt.args.indexes)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusVersion(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "returns CR PrometheusVersion when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						PrometheusVersion: "test-version",
					}
				}),
			},
			want: "test-version",
		},
		{
			name: "returns default Prometheus version when NOT self contained",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: PrometheusVersion,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusVersion(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusResourceRequirement(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want corev1.ResourceRequirements
	}{
		{
			name: "returns CR PrometheusResourceRequirement when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						PrometheusResourceRequirement: corev1.ResourceRequirements{
							Limits: testResourceList,
						},
					}
				}),
			},
			want: corev1.ResourceRequirements{
				Limits: testResourceList,
			},
		},
		{
			name: "returns blank ResourceRequirements when NOT self contained",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: corev1.ResourceRequirements{},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusResourceRequirement(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusOperatorResourceRequirement(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want corev1.ResourceRequirements
	}{
		{
			name: "returns CR PrometheusOperatorResourceRequirement when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						PrometheusOperatorResourceRequirement: corev1.ResourceRequirements{
							Limits: testResourceList,
						},
					}
				}),
			},
			want: corev1.ResourceRequirements{
				Limits: testResourceList,
			},
		},
		{
			name: "returns blank ResourceRequirements when NOT self contained",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: corev1.ResourceRequirements{},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusOperatorResourceRequirement(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusResources_GetPrometheusStorageSize(t *testing.T) {
	type args struct {
		cr      *v1.Observability
		indexes []v1.RepositoryIndex
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "cr storage is used when selfcontained is specified AND a storage value is provided",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
					obsCR.Spec.Storage = &v1.Storage{
						PrometheusStorageSpec: &monitoringv1.StorageSpec{
							VolumeClaimTemplate: monitoringv1.EmbeddedPersistentVolumeClaim{
								Spec: corev1.PersistentVolumeClaimSpec{
									Resources: corev1.ResourceRequirements{
										Requests: testResourceList,
									},
								},
							},
						},
					}
				}),
			},
			want: "10Gi",
		},

		{
			name: "default storage is used when selfcontained is NOT specified AND NO storage value is provided",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: "250Gi",
		},
		{
			name: "no nil failure when selfcontained is specified AND a storage value is NOT provided",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
				}),
			},
			want: "250Gi",
		},
		{
			name: "no nil failure when selfcontained is specified AND PersistentVolumeClaim is NOT provided",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
					obsCR.Spec.Storage = &v1.Storage{
						PrometheusStorageSpec: &monitoringv1.StorageSpec{
							VolumeClaimTemplate: monitoringv1.EmbeddedPersistentVolumeClaim{
								Spec: corev1.PersistentVolumeClaimSpec{},
							},
						},
					}
				}),
			},
			want: "250Gi",
		},
		{
			name: "no nil failure when selfcontained is specified AND EmbeddedPersistentVolumeClaim is NOT provided",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
					obsCR.Spec.Storage = &v1.Storage{
						PrometheusStorageSpec: &monitoringv1.StorageSpec{},
					}
				}),
			},
			want: "250Gi",
		},
		{
			name: "returns repo PVC override size if NOT self contained and OverridePrometheusPvcSize is not empty",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: testRepoIndexes,
			},
			want: "test-quantity",
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrometheusStorageSize(tt.args.cr, tt.args.indexes)
			Expect(result).To(Equal(tt.want))
		})
	}
}
