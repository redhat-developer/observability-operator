package grafana_configuration

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/model"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
	"github.com/jeremyary/observability-operator/controllers/utils"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	v14 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	v12 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	status, err := r.reconileProxySecret(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.reconcileClusterRole(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.reconcileClusterRoleBinding(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.reconcileGrafanaCr(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	// Grafana CR
	grafana := model.GetGrafanaCr(cr)
	err := r.client.Delete(ctx, grafana)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	status, err := r.waitForGrafanaToBeRemoved(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// Proxy Secret
	secret := model.GetGrafanaProxySecret(cr)
	err = r.client.Delete(ctx, secret)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	// Role
	clusterRoleBinding := model.GetGrafanaClusterRoleBinding(cr)
	err = r.client.Delete(ctx, clusterRoleBinding)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	clusterRole := model.GetGrafanaClusterRole(cr)
	err = r.client.Delete(ctx, clusterRole)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) waitForGrafanaToBeRemoved(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	list := &v14.DeploymentList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := r.client.List(ctx, list, opts)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	for _, ss := range list.Items {
		if ss.Name == "grafana-deployment" {
			return v1.ResultInProgress, nil
		}
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconileProxySecret(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	secret := model.GetGrafanaProxySecret(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, secret, func() error {
		if secret.Data == nil {
			secret.StringData = map[string]string{
				"session_secret": utils.GenerateRandomString(32),
			}
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileClusterRole(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	clusterRole := model.GetGrafanaClusterRole(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, clusterRole, func() error {
		clusterRole.Rules = []v12.PolicyRule{
			{
				Verbs:     []string{"create"},
				APIGroups: []string{"authentication.k8s.io"},
				Resources: []string{"tokenreviews"},
			},
			{
				Verbs:     []string{"create"},
				APIGroups: []string{"authorization.k8s.io"},
				Resources: []string{"subjectaccessreviews"},
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
	clusterRoleBinding := model.GetGrafanaClusterRoleBinding(cr)
	clusterRole := model.GetGrafanaClusterRole(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, clusterRoleBinding, func() error {
		clusterRoleBinding.RoleRef = v12.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     bundle.ClusterRoleKind,
			Name:     clusterRole.Name,
		}
		clusterRoleBinding.Subjects = []v12.Subject{
			{
				Kind:      v12.ServiceAccountKind,
				Name:      "grafana-serviceaccount", // Created by the Grafana Operator
				Namespace: cr.Namespace,
			},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileGrafanaCr(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	grafana := model.GetGrafanaCr(cr)

	var f = false
	var t = true

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, grafana, func() error {
		grafana.Spec = v1alpha1.GrafanaSpec{
			Config: v1alpha1.GrafanaConfig{
				Log: &v1alpha1.GrafanaConfigLog{
					Mode:  "console",
					Level: "warn",
				},
				Auth: &v1alpha1.GrafanaConfigAuth{
					DisableLoginForm:   &f,
					DisableSignoutMenu: &t,
				},
				AuthBasic: &v1alpha1.GrafanaConfigAuthBasic{
					Enabled: &t,
				},
				AuthAnonymous: &v1alpha1.GrafanaConfigAuthAnonymous{
					Enabled: &t,
				},
			},
			Containers: []core.Container{
				{
					Name:  "grafana-proxy",
					Image: "quay.io/openshift/origin-oauth-proxy:4.2",
					Args: []string{
						"-provider=openshift",
						"-pass-basic-auth=false",
						"-https-address=:9091",
						"-http-address=",
						"-email-domain=*",
						"-upstream=http://localhost:3000",
						"-openshift-sar={\"resource\": \"namespaces\", \"verb\": \"get\"}",
						"-openshift-delegate-urls={\"/\": {\"resource\": \"namespaces\", \"verb\": \"get\"}}",
						"-tls-cert=/etc/tls/private/tls.crt",
						"-tls-key=/etc/tls/private/tls.key",
						"-client-secret-file=/var/run/secrets/kubernetes.io/serviceaccount/token",
						"-cookie-secret-file=/etc/proxy/secrets/session_secret",
						"-openshift-service-account=grafana-serviceaccount",
						"-openshift-ca=/etc/pki/tls/cert.pem",
						"-openshift-ca=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
						"-skip-auth-regex=^/metrics",
					},
					Ports: []core.ContainerPort{
						{
							Name:          "grafana-proxy",
							ContainerPort: 9091,
						},
					},
					Resources: core.ResourceRequirements{},
					VolumeMounts: []core.VolumeMount{
						{
							Name:      "secret-grafana-k8s-tls",
							ReadOnly:  false,
							MountPath: "/etc/tls/private",
						},
						{
							Name:      "secret-grafana-k8s-proxy",
							ReadOnly:  false,
							MountPath: "/etc/proxy/secrets",
						},
					},
				},
			},
			DashboardLabelSelector: []*v13.LabelSelector{
				{
					MatchLabels: map[string]string{
						"app": "strimzi",
					},
				},
			},
			Ingress: &v1alpha1.GrafanaIngress{
				Enabled:     true,
				TargetPort:  "grafana-proxy",
				Termination: "reencrypt",
			},
			Secrets: []string{
				"grafana-k8s-tls",
				"grafana-k8s-proxy",
			},
			Service: &v1alpha1.GrafanaService{
				Annotations: map[string]string{
					"service.alpha.openshift.io/serving-cert-secret-name": "grafana-k8s-tls",
				},
				Ports: []core.ServicePort{
					{
						Name:       "grafana-proxy",
						Protocol:   "TCP",
						Port:       9091,
						TargetPort: intstr.FromString("grafana-proxy"),
					},
				},
			},
			ServiceAccount: &v1alpha1.GrafanaServiceAccount{
				Annotations: map[string]string{
					"serviceaccounts.openshift.io/oauth-redirectreference.primary": "{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"grafana-route\"}}",
				},
			},
			Client: &v1alpha1.GrafanaClient{
				PreferService: true,
			},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}
