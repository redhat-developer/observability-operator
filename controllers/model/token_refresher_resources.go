package model

import (
	"fmt"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	v13 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TokenRefresherType string

const (
	MetricsTokenRefresher TokenRefresherType = "metrics"
	LogsTokenRefresher    TokenRefresherType = "logs"
)

type TokenRefresherConfigSet struct {
	ObservatoriumUrl string
	AuthUrl          string
	Name             string
	Realm            string
	Client           string
	Tenant           string
	Secret           string
}

func GetTokenRefresherName(id string, t TokenRefresherType) string {
	return fmt.Sprintf("token-refresher-%v-%v", t, id)
}

func GetTokenRefresherService(cr *v1.Observability, name string) *v12.Service {
	return &v12.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
		},
	}
}

func GetTokenRefresherDeployment(cr *v1.Observability, name string) *v13.Deployment {
	return &v13.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
		},
	}
}
