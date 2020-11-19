package prometheus_rules

import (
	"context"
	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
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
	status, err := r.reconcileRule(ctx, cr)
	if status != v1.ResultSuccess {
		if err != nil {
			r.logger.Error(err, "error reconciling prometheus rules")
		}
		return status, err
	}
	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileRule(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	rule := &prometheusv1.PrometheusRule{
		ObjectMeta: v12.ObjectMeta{
			Name:      "kafka-prometheus-rules",
			Namespace: cr.Spec.ClusterMonitoringNamespace,
			Labels:    map[string]string{"app": "strimzi"},
		},
	}

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
