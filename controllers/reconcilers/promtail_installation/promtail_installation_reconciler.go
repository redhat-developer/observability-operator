package promtail_installation

import (
	"context"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/model"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/reconcilers"
	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
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

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	// Without Observatorium there is no need to install Promtail, because we're not
	// running on cluster Loki
	if cr.ObservatoriumDisabled() || cr.ExternalSyncDisabled() {
		return v1.ResultSuccess, nil
	}

	rolebinding := model.GetPromtailClusterRoleBinding(cr)
	err := r.client.Delete(ctx, rolebinding)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	role := model.GetPromtailClusterRole(cr)
	err = r.client.Delete(ctx, role)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	sa := model.GetPromtailServiceAccount(cr)
	err = r.client.Delete(ctx, sa)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	// Without Observatorium there is no need to install Promtail, because we're not
	// running on cluster Loki
	if cr.ObservatoriumDisabled() || cr.ExternalSyncDisabled() {
		return v1.ResultSuccess, nil
	}

	status, err := r.reconcilePromtailServiceAccount(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.reconcilePromtailClusterRole(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.reconcilePromtailClusterRoleBinding(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcilePromtailServiceAccount(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	sa := model.GetPromtailServiceAccount(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, sa, func() error {
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcilePromtailClusterRole(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	role := model.GetPromtailClusterRole(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, role, func() error {
		role.Rules = []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list", "watch"},
				APIGroups: []string{""},
				Resources: []string{
					"nodes",
					"nodes/proxy",
					"services",
					"endpoints",
					"pods",
				},
			},
			{
				Verbs:         []string{"use"},
				APIGroups:     []string{"security.openshift.io"},
				Resources:     []string{"securitycontextconstraints"},
				ResourceNames: []string{"privileged"},
			},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcilePromtailClusterRoleBinding(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	rolebinding := model.GetPromtailClusterRoleBinding(cr)
	sa := model.GetPromtailServiceAccount(cr)
	role := model.GetPromtailClusterRole(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, rolebinding, func() error {
		rolebinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     role.Name,
		}
		rolebinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: sa.Namespace,
			},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}
