package prometheus_installation

import (
	"context"
	"github.com/go-logr/logr"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/model"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
	coreosv1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v12 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciler struct {
	client client.Client
	logger logr.Logger
	scheme *runtime.Scheme
}

func NewReconciler(client client.Client, logger logr.Logger, scheme *runtime.Scheme) reconcilers.ObservabilityReconciler {
	return &Reconciler{
		client: client,
		logger: logger,
		scheme: scheme,
	}
}

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	// Delete subscription
	subscription := model.GetPrometheusSubscription(cr)
	err := r.client.Delete(ctx, subscription)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	// Delete operatorgroup
	operatorgroup := model.GetPrometheusOperatorgroup(cr)
	err = r.client.Delete(ctx, operatorgroup)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	// We have to remove the prometheus operator deployment manually
	deployments := &v12.DeploymentList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err = r.client.List(ctx, deployments, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	for _, deployment := range deployments.Items {
		if deployment.Name == "prometheus-operator" {
			err = r.client.Delete(ctx, &deployment)
			if err != nil && !errors.IsNotFound(err) {
				return v1.ResultFailed, err
			}
		}
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	// Prometheus subscription
	status, err := r.reconcileSubscription(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// Observability operator group
	status, err = r.reconcileOperatorgroup(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.waitForPrometheusOperator(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) waitForPrometheusOperator(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	// We have to remove the prometheus operator deployment manually
	deployments := &v12.DeploymentList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := r.client.List(ctx, deployments, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	for _, deployment := range deployments.Items {
		if deployment.Name == "prometheus-operator" {
			if deployment.Status.ReadyReplicas > 0 {
				return v1.ResultSuccess, nil
			}
		}
	}
	return v1.ResultInProgress, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	subscription := model.GetPrometheusSubscription(cr)

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
	operatorgroup := model.GetPrometheusOperatorgroup(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, operatorgroup, func() error {
		operatorgroup.Spec = coreosv1.OperatorGroupSpec{
			TargetNamespaces: []string{cr.Namespace},
		}

		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}
