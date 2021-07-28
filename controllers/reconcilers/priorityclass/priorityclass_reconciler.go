package priorityclass

import (
	"context"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/model"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/reconcilers"
	"github.com/go-logr/logr"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, status *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	// priorityClass := model.GetPriorityClass(cr)

	// TODO

	return v12.StatusSuccess, nil
}

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	priorityClass := model.GetPriorityClass(cr)

	err := r.client.Delete(ctx, priorityClass)
	if err != nil {
		return v12.StatusFailure, err
	}

	return v12.StatusSuccess, nil
}
