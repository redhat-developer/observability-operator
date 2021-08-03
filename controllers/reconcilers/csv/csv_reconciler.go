package csv

import (
	"context"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/model"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/reconcilers"
	"github.com/go-logr/logr"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
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

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	list := &v1alpha1.ClusterServiceVersionList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := r.client.List(ctx, list, opts)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	for _, csv := range list.Items {

		// Grafana Operator CSV
		if csv.Namespace == cr.Namespace && strings.HasPrefix(csv.Name, "grafana-operator.") {

			for i, deploymentSpec := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
				if deploymentSpec.Name == "grafana-operator" {
					// Update priority class name on the CSV
					csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[i].Spec.Template.Spec.PriorityClassName = model.ObservabilityPriorityClassName

					err := r.client.Update(ctx, &csv)
					if err != nil {
						return v1.ResultFailed, err
					}
				}
			}
		}

		// Prometheus Operator CSV
		if csv.Namespace == cr.Namespace && strings.HasPrefix(csv.Name, "prometheusoperator.") {
			for i, deploymentSpec := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
				if deploymentSpec.Name == "prometheus-operator" {
					// Update priority class name on the CSV
					csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[i].Spec.Template.Spec.PriorityClassName = model.ObservabilityPriorityClassName

					err := r.client.Update(ctx, &csv)
					if err != nil {
						return v1.ResultFailed, err
					}
				}
			}
		}
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	list := &v1alpha1.ClusterServiceVersionList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := r.client.List(ctx, list, opts)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	for _, csv := range list.Items {
		if csv.Namespace == cr.Namespace && strings.HasPrefix(csv.Name, "grafana-operator.") {
			err := r.client.Delete(ctx, &csv)
			if err != nil && !errors.IsNotFound(err) {
				return v1.ResultFailed, err
			}
			return v1.ResultInProgress, nil
		} else if csv.Namespace == cr.Namespace && strings.HasPrefix(csv.Name, "prometheusoperator.") {
			err := r.client.Delete(ctx, &csv)
			if err != nil && !errors.IsNotFound(err) {
				return v1.ResultFailed, err
			}
			return v1.ResultInProgress, nil
		}
	}

	return v1.ResultSuccess, nil
}
