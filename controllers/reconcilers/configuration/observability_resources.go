package configuration

import (
	"context"

	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	"github.com/redhat-developer/observability-operator/v4/controllers/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) ReconcileResourcesDeployment(ctx context.Context, cr *v1.Observability, image string) error {
	deployment := model.GetResourcesDeployment(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, deployment, func() error {
		deployment.Labels = map[string]string{
			"managed-by": "observability-operator",
		}
		deployment.Spec = appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"managed-by": "observability-operator",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"managed-by": "observability-operator",
					},
				},
				Spec: corev1.PodSpec{
					PriorityClassName: model.ObservabilityPriorityClassName,
					Containers: []corev1.Container{
						{
							Name:  model.GetResourcesDefaultName(cr),
							Image: image,

							ImagePullPolicy: corev1.PullAlways,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8080,
								},
							},
							Command: []string{
								"python3",
								"server.py",
							},
						},
					},

					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "rhoas-image-pull-secret",
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) ReconcileResourcesService(ctx context.Context, cr *v1.Observability) error {
	service := model.GetResourcesService(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, service, func() error {
		service.Name = model.GetResourcesDefaultName(cr)
		service.Spec.Ports = []corev1.ServicePort{
			{
				Protocol: "TCP",
				Port:     8080,
				TargetPort: intstr.IntOrString{
					IntVal: 8080,
				},
			},
		}
		service.Spec.Selector = map[string]string{
			"managed-by": "observability-operator",
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
