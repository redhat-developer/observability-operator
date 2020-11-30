package alertmanager_installation

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/model"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
	"github.com/jeremyary/observability-operator/controllers/utils"
	v14 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	status, pagerDutySecret, err := r.getPagerDutySecret(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, deadMansSnitchUrl, err := r.getDeadMansSnitchUrl(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.reconcileAlertmanagerSecret(ctx, cr, pagerDutySecret, deadMansSnitchUrl)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.reconcileAlertmanager(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	alertmanager := model.GetAlertmanagerCr(cr)
	err := r.client.Delete(ctx, alertmanager)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	status, err := r.waitForAlertmanagerToBeRemoved(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	secret := model.GetAlertmanagerSecret(cr)
	err = r.client.Delete(ctx, secret)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) waitForAlertmanagerToBeRemoved(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	list := &v14.StatefulSetList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := r.client.List(ctx, list, opts)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	alertmanager := model.GetAlertmanagerCr(cr)

	for _, ss := range list.Items {
		if ss.Name == fmt.Sprintf("prometheus-%s", alertmanager.Name) {
			return v1.ResultInProgress, nil
		}
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileAlertmanager(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	alertmanager := model.GetAlertmanagerCr(cr)
	configSecret := model.GetAlertmanagerSecret(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, alertmanager, func() error {
		alertmanager.Spec.ConfigSecret = configSecret.Name
		alertmanager.Spec.ListenLocal = true
		return nil
	})
	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, err

}

func (r *Reconciler) reconcileAlertmanagerServiceAccount(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	sa := model.GetAlertmanagerServiceAccount(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, sa, func() error {
		return nil
	})
	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, err
}

func (r *Reconciler) reconeileAlertmanagerService(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {

}


func (r *Reconciler) reconcileAlertmanagerRoute(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	route := model.GetAlertmanagerRoute(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, route, func() error {
		return nil
	})
	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, err
}

func (r *Reconciler) reconcileAlertmanagerProxySecret(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	secret := model.GetAlertmanagerProxySecret(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, secret, func() error {
		if secret.Data == nil {
			secret.Type = v12.SecretTypeOpaque
			secret.StringData = map[string]string{
				"session_secret": utils.GenerateRandomString(64),
			}
		}
		return nil
	})
	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, err

}

func (r *Reconciler) reconcileAlertmanagerSecret(ctx context.Context, cr *v1.Observability, pagerDutySecret []byte, deadMansSnitchUrl []byte) (v1.ObservabilityStageStatus, error) {
	secret := model.GetAlertmanagerSecret(cr)
	config, err := model.GetAlertmanagerConfig(cr, string(pagerDutySecret), string(deadMansSnitchUrl))
	if err != nil {
		return v1.ResultFailed, err
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, secret, func() error {
		if secret.Data == nil {
			secret.Type = v12.SecretTypeOpaque
			secret.StringData = map[string]string{
				"alertmanager.yaml": config,
			}
		}
		return nil
	})
	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, err
}

func (r *Reconciler) getPagerDutySecret(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, []byte, error) {
	if cr.Spec.Alertmanager == nil {
		return v1.ResultSuccess, nil, nil
	}

	if cr.Spec.Alertmanager.PagerDutySecretName == "" {
		return v1.ResultSuccess, nil, nil
	}

	ns := cr.Namespace
	if cr.Spec.Alertmanager.PagerDutySecretNamespace != "" {
		ns = cr.Spec.Alertmanager.PagerDutySecretNamespace
	}

	pagerdutySecret := &v12.Secret{}
	selector := client.ObjectKey{
		Namespace: ns,
		Name:      cr.Spec.Alertmanager.PagerDutySecretName,
	}
	err := r.client.Get(ctx, selector, pagerdutySecret)
	if err != nil {
		return v1.ResultFailed, nil, err
	}

	var secret []byte
	if len(pagerdutySecret.Data["PAGERDUTY_KEY"]) != 0 {
		secret = pagerdutySecret.Data["PAGERDUTY_KEY"]
	} else if len(pagerdutySecret.Data["serviceKey"]) != 0 {
		secret = pagerdutySecret.Data["serviceKey"]
	}

	return v1.ResultSuccess, secret, nil
}

func (r *Reconciler) getDeadMansSnitchUrl(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, []byte, error) {
	if cr.Spec.Alertmanager == nil {
		return v1.ResultSuccess, nil, nil
	}

	if cr.Spec.Alertmanager.DeadMansSnitchSecretName == "" {
		return v1.ResultSuccess, nil, nil
	}

	ns := cr.Namespace
	if cr.Spec.Alertmanager.DeadMansSnitchSecretNamespace != "" {
		ns = cr.Spec.Alertmanager.DeadMansSnitchSecretNamespace
	}

	dmsSecret := &v12.Secret{}
	selector := client.ObjectKey{
		Namespace: ns,
		Name:      cr.Spec.Alertmanager.DeadMansSnitchSecretName,
	}
	err := r.client.Get(ctx, selector, dmsSecret)
	if err != nil {
		return v1.ResultFailed, nil, err
	}

	var url []byte
	if len(dmsSecret.Data["SNITCH_URL"]) != 0 {
		url = dmsSecret.Data["SNITCH_URL"]
	} else if len(dmsSecret.Data["url"]) != 0 {
		url = dmsSecret.Data["url"]
	}

	return v1.ResultSuccess, url, nil
}
