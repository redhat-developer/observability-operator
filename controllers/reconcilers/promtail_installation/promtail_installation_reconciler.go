package promtail_installation

import (
	"context"
	"github.com/go-logr/logr"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/model"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
	v13 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v14 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	daemonset := model.GetPromtailDaemonSet(cr)
	err := r.client.Delete(ctx, daemonset)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	rolebinding := model.GetPromtailClusterRoleBinding(cr)
	err = r.client.Delete(ctx, rolebinding)
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

	config := model.GetPromtailConfigmap(cr)
	err = r.client.Delete(ctx, config)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	status, _, err := r.getScrapeNamespaces(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.reconcilePromtailServiceAccount(ctx, cr)
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

	status, err = r.reconcilePromtailDaemonSet(ctx, cr, "")
	if status != v1.ResultSuccess {
		return status, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) getScrapeNamespaces(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, []string, error) {
	var namespaces []string
	return v1.ResultSuccess, namespaces, nil
}

func (r *Reconciler) reconcilePromtailDaemonSet(ctx context.Context, cr *v1.Observability, hash string) (v1.ObservabilityStageStatus, error) {
	daemonset := model.GetPromtailDaemonSet(cr)
	sa := model.GetPromtailServiceAccount(cr)
	config := model.GetPromtailConfigmap(cr)
	tokenSecret := model.GetTokenSecret(cr)

	var t = true

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, daemonset, func() error {
		daemonset.Spec = v13.DaemonSetSpec{
			Selector: &v14.LabelSelector{
				MatchLabels: model.GetResourceLabels(),
			},
			Template: v12.PodTemplateSpec{
				ObjectMeta: v14.ObjectMeta{
					Labels: model.GetResourceLabels(),
				},
				Spec: v12.PodSpec{
					Affinity: &v12.Affinity{
						NodeAffinity: &v12.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v12.NodeSelector{
								NodeSelectorTerms: []v12.NodeSelectorTerm{
									{
										MatchExpressions: []v12.NodeSelectorRequirement{
											{
												Key:      "node-role.kubernetes.io/infra",
												Operator: "DoesNotExist",
											},
										},
									},
								},
							},
						},
					},
					ServiceAccountName: sa.Name,
					Volumes: []v12.Volume{
						{
							Name: "config",
							VolumeSource: v12.VolumeSource{
								ConfigMap: &v12.ConfigMapVolumeSource{
									LocalObjectReference: v12.LocalObjectReference{
										Name: config.Name,
									},
								},
							},
						},
						{
							Name: "token",
							VolumeSource: v12.VolumeSource{
								Secret: &v12.SecretVolumeSource{
									SecretName: tokenSecret.Name,
								},
							},
						},
						{
							Name: "logs",
							VolumeSource: v12.VolumeSource{
								HostPath: &v12.HostPathVolumeSource{
									Path: "/var/log/pods",
								},
							},
						},
					},
					Containers: []v12.Container{
						{
							Name:  "promtail",
							Image: "quay.io/integreatly/promtail:latest",
							SecurityContext: &v12.SecurityContext{
								Privileged: &t,
							},
							Env: []v12.EnvVar{
								{
									Name: "HOSTNAME",
									ValueFrom: &v12.EnvVarSource{
										FieldRef: &v12.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "CONFIG_HASH",
									Value: hash,
								},
							},
							Args: []string{
								"-config.file=/opt/config/promtail.yaml",
							},
							VolumeMounts: []v12.VolumeMount{
								{
									Name:      "config",
									MountPath: "/opt/config",
								},
								{
									Name:      "token",
									MountPath: "/opt/secrets",
								},
								{
									Name:      "logs",
									MountPath: "/var/log/pods",
								},
							},
							Ports: []v12.ContainerPort{
								{
									ContainerPort: 3100,
									Protocol:      "TCP",
								},
							},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: "File",
							ImagePullPolicy:          "Always",
						},
					},
				},
			},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
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
