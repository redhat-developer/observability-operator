package prometheus_configuration

import (
	"context"
	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
	"github.com/jeremyary/observability-operator/controllers/utils"
	routev1 "github.com/openshift/api/route/v1"
	core "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciler struct {
	client client.Client
	logger logr.Logger
}

func NewReconciler(client client.Client, logger logr.Logger) reconcilers.ObservabilityReconciler {
	return &Reconciler{
		client: client,
		logger: logger,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	// prometheus service account
	status, err := r.reconcileServiceAccount(ctx, cr)
	if status != v1.ResultSuccess {
		if err != nil {
			r.logger.Error(err, "error reconciling service account")
		}
		return status, err
	}

	// prometheus cluster role
	status, err = r.reconcileClusterRole(ctx)
	if status != v1.ResultSuccess {
		if err != nil {
			r.logger.Error(err, "error reconciling cluster role")
		}
		return status, err
	}

	// prometheus cluster role binding
	status, err = r.reconcileClusterRoleBinding(ctx, cr)
	if status != v1.ResultSuccess {
		if err != nil {
			r.logger.Error(err, "error reconciling cluster role binding")
		}
		return status, err
	}

	// prometheus route
	status, err = r.reconcileRoute(ctx, cr)
	if status != v1.ResultSuccess {
		if err != nil {
			r.logger.Error(err, "error reconciling route")
		}
		return status, err
	}

	// additional scrape config secret
	status, err = r.reconcileSecret(ctx, cr)
	if status != v1.ResultSuccess {
		if err != nil {
			r.logger.Error(err, "error reconciling scrape config secret")
		}
		return status, err
	}

	// prometheus instance CR
	status, err = r.reconcilePrometheus(ctx, cr)
	if status != v1.ResultSuccess {
		if err != nil {
			r.logger.Error(err, "error reconciling Prometheus CR")
		}
		return status, err
	}

	// strimzi PodMonitor
	status, err = r.reconcileStrimziPodMonitor(ctx, cr)
	if status != v1.ResultSuccess {
		if err != nil {
			r.logger.Error(err, "error reconciling Strimzi PodMonitor")
		}
		return status, err
	}

	// kafka PodMonitor
	status, err = r.reconcileKafkaPodMonitor(ctx, cr)
	if status != v1.ResultSuccess {
		if err != nil {
			r.logger.Error(err, "error reconciling Kafka PodMonitor")
		}
		return status, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileServiceAccount(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	serviceAccount := &core.ServiceAccount{
		ObjectMeta: v12.ObjectMeta{
			Name:      "kafka-prometheus",
			Namespace: cr.Spec.ClusterMonitoringNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, serviceAccount, func() error { return nil })

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileClusterRole(ctx context.Context) (v1.ObservabilityStageStatus, error) {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: v12.ObjectMeta{
			Name: "kafka-prometheus",
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, clusterRole, func() error {
		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list", "watch"},
				APIGroups: []string{""},
				Resources: []string{"services", "endpoints", "pods"},
			},
			{
				Verbs:     []string{"create"},
				APIGroups: []string{"authorization.k8s.io"},
				Resources: []string{"subjectaccessreviews"},
			},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileClusterRoleBinding(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: v12.ObjectMeta{
			Name: "kafka-prometheus",
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, clusterRoleBinding, func() error {
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "kafka-prometheus",
		}
		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "kafka-prometheus",
				Namespace: cr.Spec.ClusterMonitoringNamespace,
			},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileRoute(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	route := &routev1.Route{
		ObjectMeta: v12.ObjectMeta{
			Name:      "kafka-prometheus",
			Namespace: cr.Spec.ClusterMonitoringNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, route, func() error {
		route.Spec = routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   "prometheus-operated",
				Weight: utils.PtrToInt32(100),
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("web"),
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileSecret(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	secret := &core.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      "additional-scrape-configs",
			Namespace: cr.Spec.ClusterMonitoringNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, secret, func() error {
		secret.Type = core.SecretTypeOpaque
		secret.StringData = map[string]string{
			"additional-scrape-config.yaml": `- job_name: openshift-monitoring-federation
  honor_labels: true
  kubernetes_sd_configs:
    - role: service
      namespaces:
        names:
          - openshift-monitoring
  scrape_interval: 30s
  metrics_path: /federate
  params:
    match[]:
      - 'console_url'
      - 'cluster_version'
      - 'ALERTS'
      - 'subscription_sync_total'
      - 'kubelet_volume_stats_used_bytes{endpoint="https-metrics",namespace!~"openshift-.*$",namespace!~"kube-.*$",namespace!="default"}'
      - 'kubelet_volume_stats_available_bytes{endpoint="https-metrics",namespace!~"openshift-.*$",namespace!~"kube-.*$",namespace!="default"}'
      - 'kubelet_volume_stats_capacity_bytes{endpoint="https-metrics",namespace!~"openshift-.*$",namespace!~"kube-.*$",namespace!="default"}'
      - '{service="kube-state-metrics"}'
      - '{service="node-exporter"}'
      - '{__name__=~"node_namespace_pod_container:.*"}'
      - '{__name__=~"node:.*"}'
      - '{__name__=~"instance:.*"}'
      - '{__name__=~"container_memory_.*"}'
      - '{__name__=~"container_cpu_.*"}'
      - '{__name__=~":node_memory_.*"}'
      - '{__name__=~"csv_.*"}'
  scheme: https
  tls_config:
    insecure_skip_verify: true
  basic_auth:
    username: <user>
    password: <pass>`,
		} //TODO ^ dynamic user/pass required here
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcilePrometheus(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	prometheus := &prometheusv1.Prometheus{
		ObjectMeta: v12.ObjectMeta{
			Name:      "kafka-prometheus",
			Namespace: cr.Spec.ClusterMonitoringNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, prometheus, func() error {
		prometheus.Spec = prometheusv1.PrometheusSpec{
			ServiceAccountName: "kafka-prometheus",
			AdditionalScrapeConfigs: &core.SecretKeySelector{
				LocalObjectReference: core.LocalObjectReference{
					Name: "additional-scrape-configs",
				},
				Key: "additional-scrape-config.yaml",
			},
			PodMonitorNamespaceSelector: &v12.LabelSelector{},
			PodMonitorSelector:          &v12.LabelSelector{},
			ExternalLabels: map[string]string{
				"cluster_id": "TODO", //TODO dynamic value here instead
			},
			RuleSelector:                    &v12.LabelSelector{},
			RuleNamespaceSelector:           &v12.LabelSelector{},
			ServiceMonitorNamespaceSelector: &v12.LabelSelector{},
			ServiceMonitorSelector:          &v12.LabelSelector{},
			RemoteWrite: []prometheusv1.RemoteWriteSpec{
				{
					URL: "", //TODO backfill once we can dynamically provide centralized collection URL
					WriteRelabelConfigs: []prometheusv1.RelabelConfig{
						{
							Action: "keep",
							Regex:  "(kafka_controller.*$|console_url$|csv_succeeded$|csv_abnormal$|cluster_version$|ALERTS$|strimzi_.*$|subscription_sync_total)",
							SourceLabels: []string{
								"__name__",
							},
						},
					},
				},
			},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileStrimziPodMonitor(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	podMonitor := &prometheusv1.PodMonitor{
		ObjectMeta: v12.ObjectMeta{
			Name:      "strimzi-metrics",
			Namespace: cr.Spec.ClusterMonitoringNamespace,
			Labels:    map[string]string{"app": "strimzi"},
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, podMonitor, func() error {
		podMonitor.Spec = prometheusv1.PodMonitorSpec{
			Selector: v12.LabelSelector{
				MatchLabels: map[string]string{"strimzi.io/kind": "cluster-operator"},
			},
			NamespaceSelector: prometheusv1.NamespaceSelector{
				Any: true,
			},
			PodMetricsEndpoints: []prometheusv1.PodMetricsEndpoint{
				{
					Path: "/metrics",
					Port: "http",
				},
			},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileKafkaPodMonitor(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	podMonitor := &prometheusv1.PodMonitor{
		ObjectMeta: v12.ObjectMeta{
			Name:      "kafka-metrics",
			Namespace: cr.Spec.ClusterMonitoringNamespace,
			Labels:    map[string]string{"app": "strimzi"},
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, podMonitor, func() error {
		podMonitor.Spec = prometheusv1.PodMonitorSpec{
			Selector: v12.LabelSelector{
				MatchExpressions: []v12.LabelSelectorRequirement{
					{
						Key:      "strimzi.io/kind",
						Operator: v12.LabelSelectorOpIn,
						Values:   []string{"Kafka", "KafkaConnect"},
					},
				},
			},
			NamespaceSelector: prometheusv1.NamespaceSelector{
				Any: true,
			},
			PodMetricsEndpoints: []prometheusv1.PodMetricsEndpoint{
				{
					Path: "/metrics",
					Port: "tcp-prometheus",
					RelabelConfigs: []*prometheusv1.RelabelConfig{
						{
							Separator:   ";",
							Regex:       "__meta_kubernetes_pod_label_(.+)",
							Replacement: "$1",
							Action:      "labelmap",
						},
						{
							SourceLabels: []string{"__meta_kubernetes_namespace"},
							Separator:    ";",
							Regex:        "(.*)",
							TargetLabel:  "namespace",
							Replacement:  "$1",
							Action:       "replace",
						},
						{
							SourceLabels: []string{"__meta_kubernetes_pod_name"},
							Separator:    ";",
							Regex:        "(.*)",
							TargetLabel:  "kubernetes_pod_name",
							Replacement:  "$1",
							Action:       "replace",
						},
						{
							SourceLabels: []string{"__meta_kubernetes_pod_node_name"},
							Separator:    ";",
							Regex:        "(.*)",
							TargetLabel:  "node_name",
							Replacement:  "$1",
							Action:       "replace",
						},
						{
							SourceLabels: []string{"__meta_kubernetes_pod_host_ip"},
							Separator:    ";",
							Regex:        "(.*)",
							TargetLabel:  "node_ip",
							Replacement:  "$1",
							Action:       "replace",
						},
					},
				},
			},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}
