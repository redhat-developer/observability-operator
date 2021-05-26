package alertmanager_installation

import (
	"context"
	"fmt"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/model"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/reconcilers"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/utils"
	"github.com/go-logr/logr"
	v13 "github.com/openshift/api/route/v1"
	v14 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	v15 "k8s.io/api/rbac/v1"
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

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	status, err := r.reconcileAlertmanagerProxySecret(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.reconcileAlertmanagerServiceAccount(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.reconcileAlertmanagerClusterRole(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.reconcileAlertmanagerClusterRoleBinding(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.reconcileAlertmanagerService(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.reconcileAlertmanagerRoute(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.waitForRoute(ctx, cr)
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

	secret = model.GetAlertmanagerProxySecret(cr)
	err = r.client.Delete(ctx, secret)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	secret = model.GetAlertmanagerTLSSecret(cr)
	err = r.client.Delete(ctx, secret)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	route := model.GetAlertmanagerRoute(cr)
	err = r.client.Delete(ctx, route)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	service := model.GetAlertmanagerService(cr)
	err = r.client.Delete(ctx, service)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	sa := model.GetAlertmanagerServiceAccount(cr)
	err = r.client.Delete(ctx, sa)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	role := model.GetAlertmanagerClusterRole()
	err = r.client.Delete(ctx, role)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	binding := model.GetAlertmanagerClusterRoleBinding()
	err = r.client.Delete(ctx, binding)
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

func (r *Reconciler) waitForRoute(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	route := model.GetAlertmanagerRoute(cr)
	selector := client.ObjectKey{
		Namespace: route.Namespace,
		Name:      route.Name,
	}

	err := r.client.Get(ctx, selector, route)
	if err != nil {
		if errors.IsNotFound(err) {
			return v1.ResultInProgress, nil
		}
		return v1.ResultFailed, err
	}

	if utils.IsRouteReady(route) {
		return v1.ResultSuccess, nil
	}

	return v1.ResultInProgress, nil
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

func (r *Reconciler) reconcileAlertmanagerClusterRole(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	role := model.GetAlertmanagerClusterRole()

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, role, func() error {
		role.Rules = []v15.PolicyRule{
			{
				Verbs:     []string{"create"},
				APIGroups: []string{"authorization.k8s.io"},
				Resources: []string{"subjectaccessreviews"},
			},
			{
				Verbs:     []string{"create"},
				APIGroups: []string{"authentication.k8s.io"},
				Resources: []string{"tokenreviews"},
			},
		}
		return nil
	})
	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, err
}

func (r *Reconciler) reconcileAlertmanagerClusterRoleBinding(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	binding := model.GetAlertmanagerClusterRoleBinding()
	role := model.GetAlertmanagerClusterRole()

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, binding, func() error {
		binding.Subjects = []v15.Subject{
			{
				Kind:      v15.ServiceAccountKind,
				Name:      model.GetAlertmanagerServiceAccount(cr).Name,
				Namespace: cr.Namespace,
			},
		}
		binding.RoleRef = v15.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     role.Name,
		}
		return nil
	})
	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, err
}

func (r *Reconciler) reconcileAlertmanagerService(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	service := model.GetAlertmanagerService(cr)
	alertmanager := model.GetAlertmanagerCr(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, service, func() error {
		service.Annotations = map[string]string{
			"service.alpha.openshift.io/serving-cert-secret-name": "alertmanager-k8s-tls",
		}
		service.Spec.Ports = []v12.ServicePort{
			{
				Name:       "web",
				Protocol:   "TCP",
				Port:       9091,
				TargetPort: intstr.FromString("proxy"),
			},
		}
		service.Spec.Selector = map[string]string{
			"alertmanager": alertmanager.Name,
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, err
}

func (r *Reconciler) reconcileAlertmanagerRoute(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	route := model.GetAlertmanagerRoute(cr)
	service := model.GetAlertmanagerService(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, route, func() error {
		route.Spec.Port = &v13.RoutePort{
			TargetPort: intstr.FromString("web"),
		}
		route.Spec.TLS = &v13.TLSConfig{
			Termination: "reencrypt",
		}
		route.Spec.To = v13.RouteTargetReference{
			Kind: "Service",
			Name: service.Name,
		}
		route.Spec.WildcardPolicy = v13.WildcardPolicyNone
		return nil
	})
	if err != nil && !errors.IsAlreadyExists(err) {
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
