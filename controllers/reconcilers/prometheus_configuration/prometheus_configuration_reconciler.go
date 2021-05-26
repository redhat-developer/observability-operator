package prometheus_configuration

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/model"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/reconcilers"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/utils"
	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	v13 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	// Delete additional scrape config
	s := model.GetPrometheusAdditionalScrapeConfig(cr)
	err := r.client.Delete(ctx, s)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}
	// Delete route
	route := model.GetPrometheusRoute(cr)
	err = r.client.Delete(ctx, route)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	// Delete Prometheus CR
	prom := model.GetPrometheus(cr)
	err = r.client.Delete(ctx, prom)
	if err != nil && !errors.IsNotFound(err) && !meta.IsNoMatchError(err) {
		return v1.ResultFailed, err
	}

	// Wait for the operator to be removed
	status, err := r.waitForPrometheusToBeRemoved(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// Delete role and rolebinding
	rb := model.GetPrometheusClusterRoleBinding()
	err = r.client.Delete(ctx, rb)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	role := model.GetPrometheusClusterRole()
	err = r.client.Delete(ctx, role)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	// Service account
	sa := model.GetPrometheusServiceAccount(cr)
	err = r.client.Delete(ctx, sa)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	// Service
	service := model.GetPrometheusService(cr)
	err = r.client.Delete(ctx, service)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	// Proxy secret account
	s = model.GetPrometheusProxySecret(cr)
	err = r.client.Delete(ctx, s)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	secret := model.GetPrometheusTLSSecret(cr)
	err = r.client.Delete(ctx, secret)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) waitForPrometheusToBeRemoved(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	list := &v13.StatefulSetList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := r.client.List(ctx, list, opts)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	prom := model.GetPrometheus(cr)

	for _, ss := range list.Items {
		if ss.Name == fmt.Sprintf("prometheus-%s", prom.Name) {
			return v1.ResultInProgress, nil
		}
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	// prometheus proxy secret
	// prometheus service account
	status, err := r.reconcilePrometheusProxySecret(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// prometheus service account
	status, err = r.reconcileServiceAccount(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// prometheus cluster role
	status, err = r.reconcileClusterRole(ctx)
	if status != v1.ResultSuccess {
		return status, err
	}

	// prometheus cluster role binding
	status, err = r.reconcileClusterRoleBinding(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// prometheus service
	status, err = r.reconcileService(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// prometheus route
	status, err = r.reconcileRoute(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.waitForRoute(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// try to obtain the cluster id
	status, err = r.fetchClusterId(ctx, cr, s)
	if status != v1.ResultSuccess {
		return status, err
	}

	// create token lifetime storage
	status, err = r.reconcileTokenLifetimeStorage(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileTokenLifetimeStorage(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	configmap := model.GetPrometheusAuthTokenLifetimes(cr)
	_, err := controllerutil.CreateOrUpdate(ctx, r.client, configmap, func() error {
		configmap.Labels = map[string]string{
			"managed-by": "observability-operator",
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileServiceAccount(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	serviceAccount := model.GetPrometheusServiceAccount(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, serviceAccount, func() error { return nil })

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcilePrometheusProxySecret(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	secret := model.GetPrometheusProxySecret(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, secret, func() error {
		if secret.Data == nil {
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

func (r *Reconciler) reconcileService(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	service := model.GetPrometheusService(cr)
	prom := model.GetPrometheus(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, service, func() error {
		service.Annotations = map[string]string{
			"service.alpha.openshift.io/serving-cert-secret-name": "prometheus-k8s-tls",
		}
		service.Spec.Selector = map[string]string{
			"prometheus": prom.Name,
		}
		service.Spec.Ports = []core.ServicePort{
			{
				Name:       "web",
				Port:       9091,
				TargetPort: intstr.FromString("proxy"),
			},
			{
				Name:       "upstream",
				Port:       9090,
				TargetPort: intstr.FromString("web"),
			},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileClusterRole(ctx context.Context) (v1.ObservabilityStageStatus, error) {
	clusterRole := model.GetPrometheusClusterRole()

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, clusterRole, func() error {
		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list", "watch"},
				APIGroups: []string{""},
				Resources: []string{"services", "endpoints", "pods"},
			},
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
			{
				Verbs:     []string{"get"},
				APIGroups: []string{""},
				Resources: []string{"configmaps", "namespaces"},
			},
			{
				Verbs:           []string{"get"},
				NonResourceURLs: []string{"/metrics"},
			},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileClusterRoleBinding(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	clusterRoleBinding := model.GetPrometheusClusterRoleBinding()
	role := model.GetPrometheusClusterRole()

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, clusterRoleBinding, func() error {
		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      model.GetPrometheusServiceAccount(cr).Name,
				Namespace: cr.Namespace,
			},
		}
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     role.Name,
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileRoute(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	route := model.GetPrometheusRoute(cr)
	service := model.GetPrometheusService(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, route, func() error {
		route.Spec = routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: service.Name,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("web"),
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
			TLS: &routev1.TLSConfig{
				Termination: "reencrypt",
			},
		}
		return nil
	})

	if err != nil && !errors.IsAlreadyExists(err) {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) waitForRoute(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	route := model.GetPrometheusRoute(cr)
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

func (r *Reconciler) getOpenshiftMonitoringCredentials(ctx context.Context) (string, string, error) {
	secret := &core.Secret{}
	selector := client.ObjectKey{
		Namespace: "openshift-monitoring",
		Name:      "grafana-datasources",
	}

	err := r.client.Get(ctx, selector, secret)
	if err != nil {
		return "", "", err
	}

	// It says yaml but it's actually json
	j := secret.Data["prometheus.yaml"]

	type datasource struct {
		BasicAuthUser     string `json:"basicAuthUser"`
		BasicAuthPassword string `json:"basicAuthPassword"`
	}

	type datasources struct {
		Sources []datasource `json:"datasources"`
	}

	ds := &datasources{}
	err = json.Unmarshal(j, ds)
	if err != nil {
		return "", "", err
	}

	return ds.Sources[0].BasicAuthUser, ds.Sources[0].BasicAuthPassword, nil
}

func (r *Reconciler) fetchClusterId(ctx context.Context, cr *v1.Observability, nextStatus *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	if cr.Status.ClusterID != "" {
		return v1.ResultSuccess, nil
	}

	if cr.Spec.ClusterID != "" {
		nextStatus.ClusterID = cr.Spec.ClusterID
		return v1.ResultSuccess, nil
	}

	clusterId, err := utils.GetClusterId(ctx, r.client)
	if err != nil {
		return v1.ResultFailed, err
	}
	nextStatus.ClusterID = clusterId

	return v1.ResultSuccess, nil
}
