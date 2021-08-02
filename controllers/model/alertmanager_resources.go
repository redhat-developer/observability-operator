package model

import (
	"fmt"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	routev1 "github.com/openshift/api/route/v1"
	v12 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v13 "k8s.io/api/core/v1"
	v14 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetAlertmanagerProxySecret(cr *v1.Observability) *v13.Secret {
	return &v13.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "alertmanager-proxy",
			Namespace: cr.Namespace,
		},
	}
}

func GetAlertmanagerTLSSecret(cr *v1.Observability) *v13.Secret {
	return &v13.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "alertmanager-k8s-tls",
			Namespace: cr.Namespace,
		},
	}
}

func GetAlertmanagerRoute(cr *v1.Observability) *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-alertmanager",
			Namespace: cr.Namespace,
		},
	}
}

func GetAlertmanagerServiceAccount(cr *v1.Observability) *v13.ServiceAccount {
	route := GetAlertmanagerRoute(cr)
	redirect := fmt.Sprintf("{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"%s\"}}", route.Name)

	return &v13.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-alertmanager",
			Namespace: cr.Namespace,
			Annotations: map[string]string{
				"serviceaccounts.openshift.io/oauth-redirectreference.primary": redirect,
			},
		},
	}
}

func GetAlertmanagerClusterRole() *v14.ClusterRole {
	return &v14.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kafka-alertmanager",
		},
	}
}

func GetAlertmanagerClusterRoleBinding() *v14.ClusterRoleBinding {
	return &v14.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kafka-alertmanager",
		},
	}
}

func GetAlertmanagerCr(cr *v1.Observability) *v12.Alertmanager {
	return &v12.Alertmanager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-alertmanager",
			Namespace: cr.Namespace,
		},
	}
}

func GetAlertmanagerSecret(cr *v1.Observability) *v13.Secret {
	alertmanager := GetAlertmanagerCr(cr)

	return &v13.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("alertmanager-%s", alertmanager.Name),
			Namespace: cr.Namespace,
		},
	}
}

func GetAlertmanagerSecretName(cr *v1.Observability) string {
	override, name := cr.HasAlertmanagerConfigSecret()
	if override {
		return name
	}

	secret := GetAlertmanagerSecret(cr)
	return secret.Name
}

func GetAlertmanagerService(cr *v1.Observability) *v13.Service {
	return &v13.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-alertmanager",
			Namespace: cr.Namespace,
		},
	}
}
