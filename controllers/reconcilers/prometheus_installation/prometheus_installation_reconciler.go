package prometheus_installation

import (
	"context"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/model"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/reconcilers"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/utils"
	"github.com/go-logr/logr"
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

	// Delete catalog source
	source := model.GetPrometheusCatalogSource(cr)
	err = r.client.Delete(ctx, source)
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

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	// Catalog source
	status, err := r.reconcileCatalogSource(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// Prometheus subscription
	status, err = r.reconcileSubscription(ctx, cr)
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

func (r *Reconciler) reconcileCatalogSource(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	source := model.GetPrometheusCatalogSource(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, source, func() error {
		source.Spec = v1alpha1.CatalogSourceSpec{
			SourceType: v1alpha1.SourceTypeGrpc,
			Image:      "quay.io/integreatly/custom-prometheus-index:1.0.0",
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	subscription := model.GetPrometheusSubscription(cr)
	source := model.GetPrometheusCatalogSource(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, subscription, func() error {
		subscription.Spec = &v1alpha1.SubscriptionSpec{
			CatalogSource:          source.Name,
			CatalogSourceNamespace: cr.Namespace,
			Package:                "prometheus",
			Channel:                "preview",
			InstallPlanApproval:    v1alpha1.ApprovalAutomatic,
		}

		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileOperatorgroup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	exists, err := utils.HasOperatorGroupForNamespace(ctx, r.client, cr.Namespace)
	if err != nil {
		return v1.ResultFailed, err
	}

	if exists {
		return v1.ResultSuccess, nil
	}

	operatorgroup := model.GetPrometheusOperatorgroup(cr)

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, operatorgroup, func() error {
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
