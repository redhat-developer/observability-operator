package prometheus_rules

import (
	"context"
	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/model"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
	"k8s.io/apimachinery/pkg/api/errors"
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

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	// delete kafka prometheus rule
	rule := model.GetKafkaPrometheusRules(cr)
	err := r.client.Delete(ctx, rule)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	// prometheus rules set
	status, err := r.reconcileRule(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}
	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileRule(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	rule := model.GetKafkaPrometheusRules(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, rule, func() error {
		rule.Spec = prometheusv1.PrometheusRuleSpec{
			Groups: []prometheusv1.RuleGroup{
				{
					Name: "kafka-api-slo",
					Rules: []prometheusv1.Rule{
						{
							Alert: "ErrorBudgetBurn_ProduceRequests",
							For:   "2m",
							Annotations: map[string]string{
								"message": "High error budget burn for Failed Produce Requests (current value: {{ $value }})",
							},
							Expr: intstr.Parse("sum(kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate5m{}) > (14.40 * (1-0.90000)) " +
								"and sum(kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate1h{}) > (14.40 * (1-0.90000))"),
							Labels: map[string]string{
								"name":     "FailedProduceRequestsPerSec",
								"severity": "critical",
							},
						},
						{
							Alert: "ErrorBudgetBurn_ProduceRequests",
							For:   "15m",
							Annotations: map[string]string{
								"message": "High error budget burn for Failed Produce Requests (current value: {{ $value }})",
							},
							Expr: intstr.Parse("sum(kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate30m{}) > (6.00 * (1-0.90000)) " +
								"and sum(kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate6h{}) > (6.00 * (1-0.90000))"),
							Labels: map[string]string{
								"name":     "FailedProduceRequestsPerSec",
								"severity": "critical",
							},
						},
						{
							Alert: "ErrorBudgetBurn_ProduceRequests",
							For:   "1h",
							Annotations: map[string]string{
								"message": "High error budget burn for Failed Produce Requests (current value: {{ $value }})",
							},
							Expr: intstr.Parse("sum(kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate2h{}) > (3.00 * (1-0.90000)) " +
								"and sum(kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate1d{}) > (3.00 * (1-0.90000))"),
							Labels: map[string]string{
								"name":     "FailedProduceRequestsPerSec",
								"severity": "warning",
							},
						},
						{
							Alert: "ErrorBudgetBurn_ProduceRequests",
							For:   "3h",
							Annotations: map[string]string{
								"message": "High error budget burn for Failed Produce Requests (current value: {{ $value }})",
							},
							Expr: intstr.Parse("sum(kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate6h{}) > (1.00 * (1-0.90000)) " +
								"and sum(kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate3d{}) > (1.00 * (1-0.90000))"),
							Labels: map[string]string{
								"name":     "FailedProduceRequestsPerSec",
								"severity": "warning",
							},
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_produce_requests_total{}[1d])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_produce_requests_total{}[1d]))"),
							Labels: map[string]string{"name": "FailedProduceRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate1d",
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_produce_requests_total{}[1h])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_produce_requests_total{}[1h]))"),
							Labels: map[string]string{"name": "FailedProduceRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate1h",
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_produce_requests_total{}[2h])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_produce_requests_total{}[2h]))"),
							Labels: map[string]string{"name": "FailedProduceRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate2h",
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_produce_requests_total{}[30m])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_produce_requests_total{}[30m]))"),
							Labels: map[string]string{"name": "FailedProduceRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate30m",
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_produce_requests_total{}[3d])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_produce_requests_total{}[3d]))"),
							Labels: map[string]string{"name": "FailedProduceRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate3d",
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_produce_requests_total{}[5m])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_produce_requests_total{}[5m]))"),
							Labels: map[string]string{"name": "FailedProduceRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate5m",
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_produce_requests_total{}[6h])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_produce_requests_total{}[6h]))"),
							Labels: map[string]string{"name": "FailedProduceRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_produce_requests_total:burnrate6h",
						},
						{
							Alert: "ErrorBudgetBurn_FetchRequests",
							For:   "2m",
							Annotations: map[string]string{
								"message": "High error budget burn for Failed Fetch Requests (current value: {{ $value }})",
							},
							Expr: intstr.Parse("sum(kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate5m{}) > (14.40 * (1-0.90000)) " +
								"and sum(kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate1h{}) > (14.40 * (1-0.90000))"),
							Labels: map[string]string{
								"name":     "FailedFetchRequestsPerSec",
								"severity": "critical",
							},
						},
						{
							Alert: "ErrorBudgetBurn_FetchRequests",
							For:   "15m",
							Annotations: map[string]string{
								"message": "High error budget burn for Failed Fetch Requests (current value: {{ $value }})",
							},
							Expr: intstr.Parse("sum(kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate30m{}) > (6.00 * (1-0.90000)) " +
								"and sum(kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate6h{}) > (6.00 * (1-0.90000))"),
							Labels: map[string]string{
								"name":     "FailedFetchRequestsPerSec",
								"severity": "critical",
							},
						},
						{
							Alert: "ErrorBudgetBurn_FetchRequests",
							For:   "1h",
							Annotations: map[string]string{
								"message": "High error budget burn for Failed Fetch Requests (current value: {{ $value }})",
							},
							Expr: intstr.Parse("sum(kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate2h{}) > (3.00 * (1-0.90000)) " +
								"and sum(kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate1d{}) > (3.00 * (1-0.90000))"),
							Labels: map[string]string{
								"name":     "FailedFetchRequestsPerSec",
								"severity": "warning",
							},
						},
						{
							Alert: "ErrorBudgetBurn_FetchRequests",
							For:   "3h",
							Annotations: map[string]string{
								"message": "High error budget burn for Failed Fetch Requests (current value: {{ $value }})",
							},
							Expr: intstr.Parse("sum(kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate6h{}) > (1.00 * (1-0.90000)) " +
								"and sum(kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate3d{}) > (1.00 * (1-0.90000))"),
							Labels: map[string]string{
								"name":     "FailedFetchRequestsPerSec",
								"severity": "warning",
							},
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_fetch_requests_total{}[1d])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_fetch_requests_total{}[1d]))"),
							Labels: map[string]string{"name": "FailedFetchRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate1d",
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_fetch_requests_total{}[1h])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_fetch_requests_total{}[1h]))"),
							Labels: map[string]string{"name": "FailedFetchRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate1h",
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_fetch_requests_total{}[2h])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_fetch_requests_total{}[2h]))"),
							Labels: map[string]string{"name": "FailedFetchRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate2h",
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_fetch_requests_total{}[30m])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_fetch_requests_total{}[30m]))"),
							Labels: map[string]string{"name": "FailedFetchRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate30m",
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_fetch_requests_total{}[3d])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_fetch_requests_total{}[3d]))"),
							Labels: map[string]string{"name": "FailedFetchRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate3d",
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_fetch_requests_total{}[5m])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_fetch_requests_total{}[5m]))"),
							Labels: map[string]string{"name": "FailedFetchRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate5m",
						},
						{
							Expr: intstr.Parse("sum(rate(kafka_server_brokertopicmetrics_failed_fetch_requests_total{}[6h])) " +
								"/ sum(rate(kafka_server_brokertopicmetrics_total_fetch_requests_total{}[6h]))"),
							Labels: map[string]string{"name": "FailedFetchRequestsPerSec"},
							Record: "kafka_server_brokertopicmetrics_failed_fetch_requests_total:burnrate6h",
						},
						{
							Alert: "ErrorBudgetBurn_Connections",
							For:   "2m",
							Annotations: map[string]string{
								"message": "High error budget burn for haproxy connection errors (current value: {{ $value }})",
							},
							Expr: intstr.Parse("sum(haproxy_server_connection_errors_total:burnrate5m) > (14.40 * (1-0.90000)) " +
								"and sum(haproxy_server_connection_errors_total:burnrate1h) > (14.40 * (1-0.90000))"),
							Labels: map[string]string{
								"name":     "FailedConnectionsPerSec",
								"severity": "critical",
							},
						},
						{
							Alert: "ErrorBudgetBurn_Connections",
							For:   "15m",
							Annotations: map[string]string{
								"message": "High error budget burn for haproxy connection errors (current value: {{ $value }})",
							},
							Expr: intstr.Parse("sum(haproxy_server_connection_errors_total:burnrate30m) > (6.00 * (1-0.90000)) " +
								"and sum(haproxy_server_connection_errors_total:burnrate6h) > (6.00 * (1-0.90000))"),
							Labels: map[string]string{
								"name":     "FailedConnectionsPerSec",
								"severity": "critical",
							},
						},
						{
							Alert: "ErrorBudgetBurn_Connections",
							For:   "1h",
							Annotations: map[string]string{
								"message": "High error budget burn for haproxy connection errors (current value: {{ $value }})",
							},
							Expr: intstr.Parse("sum(haproxy_server_connection_errors_total:burnrate2h) > (3.00 * (1-0.90000)) " +
								"and sum(haproxy_server_connection_errors_total:burnrate1d) > (3.00 * (1-0.90000))"),
							Labels: map[string]string{
								"name":     "FailedConnectionsPerSec",
								"severity": "warning",
							},
						},
						{
							Alert: "ErrorBudgetBurn_Connections",
							For:   "3h",
							Annotations: map[string]string{
								"message": "High error budget burn for haproxy connection errors (current value: {{ $value }})",
							},
							Expr: intstr.Parse("sum(haproxy_server_connection_errors_total:burnrate6h) > (1.00 * (1-0.90000)) " +
								"and sum(haproxy_server_connection_errors_total:burnrate3d) > (1.00 * (1-0.90000))"),
							Labels: map[string]string{
								"name":     "FailedConnectionsPerSec",
								"severity": "warning",
							},
						},
						{
							Expr: intstr.Parse("sum(rate(haproxy_server_connection_errors_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[1d])) " +
								"/ sum(rate(haproxy_server_connections_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[1d]))"),
							Labels: map[string]string{"name": "FailedConnectionsPerSec"},
							Record: "haproxy_server_connection_errors_total:burnrate1d",
						},
						{
							Expr: intstr.Parse("sum(rate(haproxy_server_connection_errors_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[1h])) " +
								"/ sum(rate(haproxy_server_connections_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[1h]))"),
							Labels: map[string]string{"name": "FailedConnectionsPerSec"},
							Record: "haproxy_server_connection_errors_total:burnrate1h",
						},
						{
							Expr: intstr.Parse("sum(rate(haproxy_server_connection_errors_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[2h])) " +
								"/ sum(rate(haproxy_server_connections_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[2h]))"),
							Labels: map[string]string{"name": "FailedConnectionsPerSec"},
							Record: "haproxy_server_connection_errors_total:burnrate2h",
						},
						{
							Expr: intstr.Parse("sum(rate(haproxy_server_connection_errors_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[30m])) " +
								"/ sum(rate(haproxy_server_connections_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[30m]))"),
							Labels: map[string]string{"name": "FailedConnectionsPerSec"},
							Record: "haproxy_server_connection_errors_total",
						},
						{
							Expr: intstr.Parse("sum(rate(haproxy_server_connection_errors_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[3d])) " +
								"/ sum(rate(haproxy_server_connections_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[3d]))"),
							Labels: map[string]string{"name": "FailedConnectionsPerSec"},
							Record: "haproxy_server_connection_errors_total:burnrate3d",
						},
						{
							Expr: intstr.Parse("sum(rate(haproxy_server_connection_errors_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[5m])) " +
								"/ sum(rate(haproxy_server_connections_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[5m]))"),
							Labels: map[string]string{"name": "FailedConnectionsPerSec"},
							Record: "haproxy_server_connection_errors_total:burnrate5m",
						},
						{
							Expr: intstr.Parse("sum(rate(haproxy_server_connection_errors_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[6h])) " +
								"/ sum(rate(haproxy_server_connections_total{route=~\".+-kafka-([0-9]+|bootstrap)$\"}[6h]))"),
							Labels: map[string]string{"name": "FailedConnectionsPerSec"},
							Record: "haproxy_server_connection_errors_total:burnrate6h",
						},
					},
				},
				{
					Name: "kafka",
					Rules: []prometheusv1.Rule{

						{
							Alert: "KafkaPersistentVolumeFillingUp",
							For:   "1m",
							Annotations: map[string]string{
								"summary": "Kafka Broker PersistentVolume is filling up.",
								"description": "The Kafka Broker PersistentVolume claimed by {{ $labels.persistentvolumeclaim }} in " +
									"Namespace {{ $labels.namespace }} is only {{ $value | humanizePercentage }} free.",
							},
							Expr: intstr.Parse("kubelet_volume_stats_available_bytes{persistentvolumeclaim=~\"data-([0-9]+)?-(.+)-kafka-[0-9]+\"} " +
								"/ kubelet_volume_stats_capacity_bytes{persistentvolumeclaim=~\"data-([0-9]+)?-(.+)-kafka-[0-9]+\"} < 0.03"),
							Labels: map[string]string{
								"severity": "critical",
							},
						},
						{
							Alert: "KafkaPersistentVolumeFillingUp",
							For:   "1h",
							Annotations: map[string]string{
								"summary":     "Kafka Broker PersistentVolume is filling up.",
								"description": "",
							},
							Expr: intstr.Parse("(kubelet_volume_stats_available_bytes{persistentvolumeclaim=~\"data-([0-9]+)?-(.+)-kafka-[0-9]+\"} " +
								"/ kubelet_volume_stats_capacity_bytes{persistentvolumeclaim=~\"data-([0-9]+)?-(.+)-kafka-[0-9]+\"} < 0.15) and " +
								"predict_linear(kubelet_volume_stats_available_bytes{persistentvolumeclaim=~\"data-([0-9]+)?-(.+)-kafka-[0-9]+\"}[6h], 4 * 24 * 3600) < 0"),
							Labels: map[string]string{
								"severity": "warning",
							},
						},
						{
							Alert: "UnderReplicatedPartitions",
							For:   "10s",
							Annotations: map[string]string{
								"summary":     "Kafka under replicated partitions",
								"description": "There are {{ $value }} under replicated partitions on {{ $labels.kubernetes_pod_name }}",
							},
							Expr: intstr.Parse("kafka_server_replicamanager_under_replicated_partitions > 0"),
							Labels: map[string]string{
								"severity": "warning",
							},
						},
						{
							Alert: "AbnormalControllerState",
							For:   "10s",
							Annotations: map[string]string{
								"summary":     "Kafka abnormal controller state",
								"description": "There are {{ $value }} active controllers in the cluster",
							},
							Expr: intstr.Parse("sum(kafka_controller_kafkacontroller_active_controller_count) != 1"),
							Labels: map[string]string{
								"severity": "warning",
							},
						},
						{
							Alert: "OfflinePartitions",
							For:   "10s",
							Annotations: map[string]string{
								"summary":     "Kafka offline partitions",
								"description": "One or more partitions have no leader",
							},
							Expr: intstr.Parse("sum(kafka_controller_kafkacontroller_offline_partitions_count) > 0"),
							Labels: map[string]string{
								"severity": "warning",
							},
						},
						{
							Alert: "UnderMinIsrPartitionCount",
							For:   "10s",
							Annotations: map[string]string{
								"summary":     "Kafka under min ISR partitions",
								"description": "There are {{ $value }} partitions under the min ISR on {{ $labels.kubernetes_pod_name }}",
							},
							Expr: intstr.Parse("kafka_server_replicamanager_under_min_isr_partition_count > 0"),
							Labels: map[string]string{
								"severity": "warning",
							},
						},
						{
							Alert: "OfflineLogDirectoryCount",
							For:   "10s",
							Annotations: map[string]string{
								"summary":     "Kafka offline log directories",
								"description": "There are {{ $value }} offline log directories on {{ $labels.kubernetes_pod_name }}",
							},
							Expr: intstr.Parse("kafka_log_logmanager_offline_log_directory_count > 0"),
							Labels: map[string]string{
								"severity": "warning",
							},
						},
						{
							Alert: "ScrapeProblem",
							For:   "3m",
							Annotations: map[string]string{
								"summary":     "Prometheus unable to scrape metrics from {{ $labels.kubernetes_pod_name }}/{{ $labels.instance }}",
								"description": "Prometheus was unable to scrape metrics from {{ $labels.kubernetes_pod_name }}/{{ $labels.instance }} for more than 3 minutes",
							},
							Expr: intstr.Parse("up{kubernetes_namespace!~\"openshift-.+\",kubernetes_pod_name=~\".+-kafka-[0-9]+\"} == 0"),
							Labels: map[string]string{
								"severity": "major",
							},
						},
						{
							Alert: "ClusterOperatorContainerDown",
							For:   "1m",
							Annotations: map[string]string{
								"summary":     "Cluster Operator down",
								"description": "The Cluster Operator has been down for longer than 90 seconds",
							},
							Expr: intstr.Parse("count((container_last_seen{container=\"strimzi-cluster-operator\"} > (time() - 90))) < 1 " +
								"or absent(container_last_seen{container=\"strimzi-cluster-operator\"})"),
							Labels: map[string]string{
								"severity": "major",
							},
						},
						{
							Alert: "KafkaBrokerContainersDown",
							For:   "3m",
							Annotations: map[string]string{
								"summary":     "All `kafka` containers down or in CrashLookBackOff status",
								"description": "All `kafka` containers have been down or in CrashLookBackOff status for 3 minutes",
							},
							Expr: intstr.Parse("absent(container_last_seen{container=\"kafka\",pod=~\".+-kafka-[0-9]+\"})"),
							Labels: map[string]string{
								"severity": "major",
							},
						},
						{
							Alert: "KafkaContainerRestartedInTheLast5Minutes",
							For:   "5m",
							Annotations: map[string]string{
								"summary":     "One or more Kafka containers restarted too often",
								"description": "One or more Kafka containers were restarted too often within the last 5 minutes",
							},
							Expr: intstr.Parse("count(count_over_time(container_last_seen{container=\"kafka\"}[5m])) > 2 * " +
								"count(container_last_seen{container=\"kafka\",pod=~\".+-kafka-[0-9]+\"})"),
							Labels: map[string]string{
								"severity": "warning",
							},
						},
					},
				},
				{
					Name: "zookeeper",
					Rules: []prometheusv1.Rule{
						{
							Alert: "AvgRequestLatency",
							For:   "10s",
							Annotations: map[string]string{
								"summary":     "Zookeeper average request latency",
								"description": "The average request latency is {{ $value }} on {{ $labels.kubernetes_pod_name }}",
							},
							Expr: intstr.Parse("zookeeper_avg_request_latency > 10"),
							Labels: map[string]string{
								"severity": "warning",
							},
						},
						{
							Alert: "OutstandingRequests",
							For:   "10s",
							Annotations: map[string]string{
								"summary":     "Zookeeper outstanding requests",
								"description": "There are {{ $value }} outstanding requests on {{ $labels.kubernetes_pod_name }}",
							},
							Expr: intstr.Parse("zookeeper_outstanding_requests > 10"),
							Labels: map[string]string{
								"severity": "warning",
							},
						},
						{
							Alert: "ZookeeperPersistentVolumeFillingUp",
							For:   "1m",
							Annotations: map[string]string{
								"summary": "Zookeeper PersistentVolume is filling up.",
								"description": "The Zookeeper PersistentVolume claimed by {{ $labels.persistentvolumeclaim }} in " +
									"Namespace {{ $labels.namespace }} is only {{ $value | humanizePercentage }} free.",
							},
							Expr: intstr.Parse("kubelet_volume_stats_available_bytes{persistentvolumeclaim=~\"data-(.+)-zookeeper-[0-9]+\"} " +
								"/ kubelet_volume_stats_capacity_bytes{persistentvolumeclaim=~\"data-(.+)-zookeeper-[0-9]+\"} < 0.03"),
							Labels: map[string]string{
								"severity": "critical",
							},
						},
						{
							Alert: "ZookeeperPersistentVolumeFillingUp",
							For:   "1h",
							Annotations: map[string]string{
								"summary": "Zookeeper PersistentVolume is filling up.",
								"description": "Based on recent sampling, the Zookeeper PersistentVolume claimed by {{ $labels.persistentvolumeclaim }} in " +
									"Namespace {{ $labels.namespace }} is expected to fill up within four days. Currently {{ $value | humanizePercentage }} is available.",
							},
							Expr: intstr.Parse("(kubelet_volume_stats_available_bytes{persistentvolumeclaim=~\"data-(.+)-zookeeper-[0-9]+\"} " +
								"/ kubelet_volume_stats_capacity_bytes{persistentvolumeclaim=~\"data-(.+)-zookeeper-[0-9]+\"} < 0.15) " +
								"and predict_linear(kubelet_volume_stats_available_bytes{persistentvolumeclaim=~\"data-(.+)-zookeeper-[0-9]+\"}[6h], 4 * 24 * 3600) < 0"),
							Labels: map[string]string{
								"severity": "",
							},
						},
						{
							Alert: "ZookeeperContainerRestartedInTheLast5Minutes",
							For:   "5m",
							Annotations: map[string]string{
								"summary": "One or more Zookeeper containers were restarted too often",
								"description": "One or more Zookeeper containers were restarted too often within the last 5 minutes. This alert can be ignored " +
									"when the Zookeeper cluster is scaling up",
							},
							Expr: intstr.Parse("count(count_over_time(container_last_seen{container=\"zookeeper\"}[5m])) " +
								"> 2 * count(container_last_seen{container=\"zookeeper\",pod=~\".+-zookeeper-[0-9]+\"})"),
							Labels: map[string]string{
								"severity": "warning",
							},
						},
						{
							Alert: "ZookeeperContainersDown",
							For:   "3m",
							Annotations: map[string]string{
								"summary":     "All `zookeeper` containers in the Zookeeper pods down or in CrashLookBackOff status",
								"description": "All `zookeeper` containers in the Zookeeper pods have been down or in CrashLookBackOff status for 3 minutes",
							},
							Expr: intstr.Parse("absent(container_last_seen{container=\"zookeeper\",pod=~\".+-zookeeper-[0-9]+\"})"),
							Labels: map[string]string{
								"severity": "major",
							},
						},
					},
				},
				{
					Name: "kafkaExporter",
					Rules: []prometheusv1.Rule{
						{
							Alert: "UnderReplicatedPartition",
							For:   "10s",
							Annotations: map[string]string{
								"summary":     "Topic has under-replicated partitions",
								"description": "Topic  {{ $labels.topic }} has {{ $value }} under-replicated partition {{ $labels.partition }}",
							},
							Expr: intstr.Parse("kafka_topic_partition_under_replicated_partition > 0"),
							Labels: map[string]string{
								"severity": "warning",
							},
						},
						{
							Alert: "TooLargeConsumerGroupLag",
							For:   "10s",
							Annotations: map[string]string{
								"summary": "Consumer group lag is too big",
								"description": "Consumer group {{ $labels.consumergroup}} lag is too big ({{ $value }}) on topic " +
									"{{ $labels.topic }}/partition {{ $labels.partition }}",
							},
							Expr: intstr.Parse("kafka_consumergroup_lag > 1000"),
							Labels: map[string]string{
								"severity": "warning",
							},
						},
						{
							Alert: "NoMessageForTooLong",
							For:   "10s",
							Annotations: map[string]string{
								"summary":     "No message for 10 minutes",
								"description": "There is no messages in topic {{ $labels.topic}}/partition {{ $labels.partition }} for 10 minutes",
							},
							Expr: intstr.Parse("changes(kafka_topic_partition_current_offset{topic!=\"__consumer_offsets\"}[10m]) == 0"),
							Labels: map[string]string{
								"severity": "warning",
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
