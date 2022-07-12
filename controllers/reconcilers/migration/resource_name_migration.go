package migration

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	"github.com/redhat-developer/observability-operator/v3/controllers/model"
	"github.com/redhat-developer/observability-operator/v3/controllers/reconcilers"
	v13 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	return v1.ResultSuccess, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	if s.Migrated {
		return v1.ResultSuccess, nil
	}

	prometheusCr := model.GetPrometheus(cr)
	prometheusCr.Name = model.PrometheusOldDefaultName

	err := r.client.Delete(ctx, prometheusCr)
	if err != nil {
		return v1.ResultFailed, err
	}

	r.waitForPrometheusToBeRemoved(ctx, cr, model.PrometheusOldDefaultName)

	prometheusRoute := model.GetPrometheusRoute(cr)
	prometheusRoute.Name = model.PrometheusOldDefaultName
	err = r.client.Delete(ctx, prometheusRoute)
	if err != nil {
		return v1.ResultFailed, err
	}

	prometheusService := model.GetPrometheusService(cr)
	prometheusService.Name = model.PrometheusOldDefaultName
	err = r.client.Delete(ctx, prometheusService)
	if err != nil {
		return v1.ResultFailed, err
	}

	prometheusServiceAccount := model.GetPrometheusServiceAccount(cr)
	prometheusServiceAccount.Name = model.PrometheusOldDefaultName
	err = r.client.Delete(ctx, prometheusServiceAccount)
	if err != nil {
		return v1.ResultFailed, err
	}

	s.Migrated = true

	return v1.ResultSuccess, nil
}

func (r *Reconciler) waitForPrometheusToBeRemoved(ctx context.Context, cr *v1.Observability, prometheusName string) (v1.ObservabilityStageStatus, error) {
	list := &v13.StatefulSetList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := r.client.List(ctx, list, opts)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	for _, ss := range list.Items {
		if ss.Name == fmt.Sprintf("prometheus-%s", prometheusName) {
			return v1.ResultInProgress, nil
		}
	}

	return v1.ResultSuccess, nil
}
