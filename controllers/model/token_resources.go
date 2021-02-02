package model

import (
	v1 "github.com/jeremyary/observability-operator/api/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetTokenSecret(cr *v1.Observability, name string) *v12.Secret {
	return &v12.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"managed-by": "observability-operator",
				"purpose":    "observatorium-token-secret",
			},
		},
	}
}
