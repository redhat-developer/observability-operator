package model

import (
	v1alpha12 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	v13 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	v14 "k8s.io/api/core/v1"
	v15 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var defaultGrafanaLabelSelectors = map[string]string{"app": "strimzi"}

const (
	GrafanaOldDefaultName   = "kafka-grafana"
	GrafanaDefaultInitImage = "quay.io/grafana-operator/grafana_plugins_init:0.1.0"
)

func GetDefaultNameGrafana(cr *v1.Observability) string {
	if cr.Spec.SelfContained != nil && cr.Spec.GrafanaDefaultName != "" {
		return cr.Spec.GrafanaDefaultName
	}
	return "obs-grafana"
}

func GetGrafanaCatalogSource(cr *v1.Observability) *v1alpha1.CatalogSource {
	return &v1alpha1.CatalogSource{
		ObjectMeta: v12.ObjectMeta{
			Name:      "grafana-operator-catalog-source",
			Namespace: cr.Namespace,
		},
	}
}

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
			Name:      GetDefaultNameGrafana(cr),
			Namespace: cr.Namespace,
		},
	}
}

func GetGrafanaDatasource(cr *v1.Observability) *v1alpha12.GrafanaDataSource {
	return &v1alpha12.GrafanaDataSource{
		ObjectMeta: v12.ObjectMeta{
			Name:      "on-cluster-prometheus",
			Namespace: cr.Namespace,
		},
	}
}

func GetGrafanaDashboardLabelSelectors(cr *v1.Observability, indexes []v1.RepositoryIndex) *v12.LabelSelector {
	// if selfcontained is set override default
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.GrafanaDashboardLabelSelector != nil {
		return cr.Spec.SelfContained.GrafanaDashboardLabelSelector
	}

	if len(indexes) > 0 {
		// We should only have one Grafana CR for the whole cluster. However, we cannot merge
		// all of the label selectors from all of the repository index config as this will result
		// in an AND requirement. Since we do not use multiple repositories on the same cluster just yet,
		// there should only be one index available in the repository index list.
		// This needs to be changed once we start using multiple repository configurations on the same cluster.
		config := indexes[0].Config
		if config != nil && config.Grafana != nil && config.Grafana.DashboardLabelSelector != nil {
			return config.Grafana.DashboardLabelSelector
		}
	}

	return &v12.LabelSelector{
		MatchLabels: defaultGrafanaLabelSelectors,
	}
}
func GetGrafanaResourceRequirement(cr *v1.Observability) *v14.ResourceRequirements {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.GrafanaResourceRequirement != nil {
		return cr.Spec.SelfContained.GrafanaResourceRequirement
	}
	return &v14.ResourceRequirements{}
}

func GetGrafanaVersion(indexes []v1.RepositoryIndex, cr *v1.Observability) string {
	if len(indexes) > 0 {
		config := indexes[0].Config
		if config != nil && config.Grafana.GrafanaVersion != "" {
			return config.Grafana.GrafanaVersion
		}
	}
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.GrafanaVersion != "" {
		return cr.Spec.SelfContained.GrafanaVersion
	}
	return ""
}

func GetGrafanaOperatorResourceRequirement(cr *v1.Observability) *v14.ResourceRequirements {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.GrafanaOperatorResourceRequirement != nil {
		return cr.Spec.SelfContained.GrafanaOperatorResourceRequirement
	}
	return &v14.ResourceRequirements{}
}

func GetGrafanaInitImage(cr *v1.Observability) string {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.GrafanaInitImage != "" {
		return cr.Spec.SelfContained.GrafanaInitImage
	}
	return GrafanaDefaultInitImage
}
