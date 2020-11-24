package model

import (
	v1alpha12 "github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	v13 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v14 "k8s.io/api/core/v1"
	v15 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetGrafanaSubscription(cr *v1.Observability) *v1alpha1.Subscription {
	return &v1alpha1.Subscription{
		ObjectMeta: v12.ObjectMeta{
			Name:      "grafana-subscription",
			Namespace: cr.Namespace,
		},
	}
}

func GetGrafanaOperatorGroup(cr *v1.Observability) *v13.OperatorGroup {
	return &v13.OperatorGroup{
		ObjectMeta: v12.ObjectMeta{
			Name:      "observability-operatorgroup",
			Namespace: cr.Namespace,
		},
	}
}

func GetGrafanaProxySecret(cr *v1.Observability) *v14.Secret {
	return &v14.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      "grafana-k8s-proxy",
			Namespace: cr.Namespace,
		},
	}
}

func GetGrafanaClusterRole(cr *v1.Observability) *v15.ClusterRole {
	return &v15.ClusterRole{
		ObjectMeta: v12.ObjectMeta{
			Name: "grafana-oauth-proxy-cluster-role",
		},
	}
}

func GetGrafanaClusterRoleBinding(cr *v1.Observability) *v15.ClusterRoleBinding {
	return &v15.ClusterRoleBinding{
		ObjectMeta: v12.ObjectMeta{
			Name: "cluster-grafana-oauth-proxy-cluster-role-binding",
		},
	}
}

func GetGrafanaCr(cr *v1.Observability) *v1alpha12.Grafana {
	return &v1alpha12.Grafana{
		ObjectMeta: v12.ObjectMeta{
			Name:      "kafka-grafana",
			Namespace: cr.Namespace,
		},
	}
}
