package model

import (
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	scheduling "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetPriorityClass(cr *v1.Observability) *scheduling.PriorityClass {
	return &scheduling.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "observability-priority-class",
			Namespace: cr.Namespace,
		},
	}
}
