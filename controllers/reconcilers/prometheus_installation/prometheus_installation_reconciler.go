package prometheus_installation

import (
	"context"
	"github.com/go-logr/logr"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
	"github.com/jeremyary/observability-operator/controllers/utils"
	coreosv1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// Namespace via project request
	status, err := r.reconcileNamespace(ctx, cr)
	if status != v1.ResultSuccess {
		if err != nil {
			r.logger.Error(err, "error reconciling namespace")
		}
		return status, err
	}

	// Prometheus subscription
	status, err = r.reconcileSubscription(ctx, cr)
	if status != v1.ResultSuccess {
		if err != nil {
			r.logger.Error(err, "error reconciling subscription")
		}
		return status, err
	}

	// Prometheus operator group
	status, err = r.reconcileOperatorgroup(ctx, cr)
	if status != v1.ResultSuccess {
		if err != nil {
			r.logger.Error(err, "error reconciling operator group")
		}
		return status, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileNamespace(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	return utils.ReconcileNamespace(ctx, r.client, cr.Spec.ClusterMonitoringNamespace)
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	subscription := &v1alpha1.Subscription{
		ObjectMeta: v12.ObjectMeta{
			Name:      "prometheus-subscription",
			Namespace: cr.Spec.ClusterMonitoringNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, subscription, func() error {
		subscription.Spec = &v1alpha1.SubscriptionSpec{
			CatalogSource:          "community-operators",
			CatalogSourceNamespace: "openshift-marketplace",
			Package:                "prometheus",
			Channel:                "beta",
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileOperatorgroup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	operatorgroup := &coreosv1.OperatorGroup{
		ObjectMeta: v12.ObjectMeta{
			Name:      "prometheus-operatorgroup",
			Namespace: cr.Spec.ClusterMonitoringNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, operatorgroup, func() error {
		operatorgroup.Spec = coreosv1.OperatorGroupSpec{
			TargetNamespaces: []string{cr.Spec.ClusterMonitoringNamespace},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}
