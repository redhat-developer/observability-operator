package csv

import (
	"context"
	"github.com/go-logr/logr"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
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

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
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
