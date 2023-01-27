package model

import (
	v12 "github.com/openshift/api/route/v1"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetResourcesDefaultName(cr *v1.Observability) string {
	return "obs-resources"
}

func GetResourcesService(cr *v1.Observability) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetResourcesDefaultName(cr),
			Namespace: cr.GetNamespace(),
		},
	}
}

func GetResourcesDeployment(cr *v1.Observability) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetResourcesDefaultName(cr),
			Namespace: cr.GetNamespace(),
		},
	}
}

func GetResourcesRoute(cr *v1.Observability) *v12.Route {
	return &v12.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "observability-resources-route",
			Namespace: cr.Namespace,
		},
	}
}
