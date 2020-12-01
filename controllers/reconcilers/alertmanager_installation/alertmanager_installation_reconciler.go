package alertmanager_installation

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/model"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
	"github.com/jeremyary/observability-operator/controllers/utils"
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

	status, err = r.reconcileAlertmanagerProxySecret(ctx, cr)
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

	status, err = r.reconcileAlertmanager(ctx, cr)
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

	if utils.IsRouteReads(route) {
		return v1.ResultSuccess, nil
	}

	return v1.ResultInProgress, nil
}

func (r *Reconciler) reconcileAlertmanager(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	alertmanager := model.GetAlertmanagerCr(cr)
	configSecret := model.GetAlertmanagerSecret(cr)
	proxySecret := model.GetAlertmanagerProxySecret(cr)
	sa := model.GetAlertmanagerServiceAccount(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, alertmanager, func() error {
		alertmanager.Spec.ConfigSecret = configSecret.Name
		alertmanager.Spec.ListenLocal = true
		alertmanager.Spec.ServiceAccountName = sa.Name
		alertmanager.Spec.Secrets = []string{
			proxySecret.Name,
			"alertmanager-k8s-tls",
		}
		alertmanager.Spec.Containers = []v12.Container{
			{
				Name:  "oauth-proxy",
				Image: "quay.io/openshift/origin-oauth-proxy:4.2",
				Args: []string{
					"-provider=openshift",
					"-https-address=:9091",
					"-http-address=",
					"-email-domain=*",
					"-upstream=http://localhost:9093",
					"-openshift-sar={\"resource\": \"namespaces\", \"verb\": \"get\"}",
					"-openshift-delegate-urls={\"/\": {\"resource\": \"namespaces\", \"verb\": \"get\"}}",
					"-tls-cert=/etc/tls/private/tls.crt",
					"-tls-key=/etc/tls/private/tls.key",
					"-client-secret-file=/var/run/secrets/kubernetes.io/serviceaccount/token",
					"-cookie-secret-file=/etc/proxy/secrets/session_secret",
					fmt.Sprintf("-openshift-service-account=%v", sa.Name),
					"-openshift-ca=/etc/pki/tls/cert.pem",
					"-openshift-ca=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
					"-skip-auth-regex=^/metrics",
				},
				Ports: []v12.ContainerPort{
					{
						Name:          "proxy",
						ContainerPort: 9091,
					},
				},
				Env: []v12.EnvVar{
					{
						Name: "HTTP_PROXY",
					},
					{
						Name: "HTTPS_PROXY",
					},
					{
						Name: "NO_PROXY",
					},
				},
				VolumeMounts: []v12.VolumeMount{
					{
						Name:      "secret-alertmanager-k8s-tls",
						MountPath: "/etc/tls/private",
					},
					{
						Name:      fmt.Sprintf("secret-%v", proxySecret.Name),
						MountPath: "/etc/proxy/secrets",
					},
				},
			},
		}
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

func (r *Reconciler) reconcileAlertmanagerSecret(ctx context.Context, cr *v1.Observability, pagerDutySecret []byte, deadMansSnitchUrl []byte) (v1.ObservabilityStageStatus, error) {
	secret := model.GetAlertmanagerSecret(cr)
	config, err := model.GetAlertmanagerConfig(string(pagerDutySecret), string(deadMansSnitchUrl))
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
