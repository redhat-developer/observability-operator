package promtail_installation

import (
	"context"
	"github.com/go-logr/logr"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/model"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
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
	return v1.ResultSuccess, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcilePromtailConfig(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	config := model.GetPromtailConfigmap(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, config, func () error {
		if config.Data == nil {
			config.Data = make(map[string]string)
		}
		config.Data["promtail.yaml"] = ""
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}