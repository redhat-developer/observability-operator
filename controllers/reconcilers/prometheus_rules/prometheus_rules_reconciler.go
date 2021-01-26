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

	rule = model.GetKafkaDeadmansSwitch(cr)
	err = r.client.Delete(ctx, rule)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	// deadmansswitch
	status, err := r.reconcileDeadmansSwitch(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// prometheus rules set
	status, err = r.reconcileRule(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}
	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileDeadmansSwitch(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	rule := model.GetKafkaDeadmansSwitch(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, rule, func() error {
		rule.Spec = prometheusv1.PrometheusRuleSpec{
			Groups: []prometheusv1.RuleGroup{
				{
					Name: "general.rules",
					Rules: []prometheusv1.Rule{
						{
							Alert: "DeadMansSwitch",
							Expr:  intstr.FromString("vector(1)"),
							Labels: map[string]string{
								"severity": "none",
							},
							Annotations: map[string]string{
								"description": "This is a DeadMansSwitch meant to ensure that the entire Alerting pipeline is functional.",
								"summary":     "Alerting DeadMansSwitch",
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

func (r *Reconciler) reconcileRule(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	return v1.ResultSuccess, nil
}
