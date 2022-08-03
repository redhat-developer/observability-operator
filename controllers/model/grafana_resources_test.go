package model

import (
	"testing"

	v1alpha12 "github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	. "github.com/onsi/gomega"
	coreosv1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	corev1 "k8s.io/api/core/v1"
	v14 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	defaultGrafanaName         = "kafka-grafana"
	objectMetaWithNamespace    = v12.ObjectMeta{Namespace: testNamespace}
	labelSelectorWithNamespace = &v12.LabelSelector{MatchLabels: map[string]string{"namespace": "test"}}
	testRepoConfig             = []v1.RepositoryIndex{
		{
			Config: &v1.RepositoryConfig{
				Grafana: &v1.GrafanaIndex{
					DashboardLabelSelector: labelSelectorWithNamespace,
					GrafanaVersion:         "8.4.1",
				},
			},
		},
	}
)

func TestGrafanaResources_GetDefaultNameGrafana(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "returns 'kafka-grafana' if NOT self contained",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: defaultGrafanaName,
		},
		{
			name: "returns CR default Grafana name if self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
					obsCR.Spec.GrafanaDefaultName = "test"
				}),
			},
			want: "test",
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDefaultNameGrafana(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaResources_GetGrafanaCatalogSource(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *v1alpha1.CatalogSource
	}{
		{
			name: "returns 'grafana-operator-catalog-source' CatalogSource",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.ObjectMeta = objectMetaWithNamespace
				}),
			},
			want: &v1alpha1.CatalogSource{
				ObjectMeta: v12.ObjectMeta{
					Name:      "grafana-operator-catalog-source",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGrafanaCatalogSource(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaResources_GetGrafanaVersion(t *testing.T) {
	type args struct {
		indexes []v1.RepositoryIndex
		cr      *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "returns empty string if no version is declared in the Repository Index",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: []v1.RepositoryIndex{},
			},
			want: "",
		},
		{
			name: "return version specified in the Repository Index",
			args: args{
				indexes: testRepoConfig,
				cr:      buildObservabilityCR(nil),
			},
			want: "8.4.1",
		},
		{
			name: "returns empty string if no version is declared in the Repository Index",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
					obsCR.Spec.SelfContained.GrafanaVersion = "8.4.1"
				}),
			},
			want: "8.4.1",
		},
		{
			name: "return version specified in SelfContained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
					obsCR.Spec.SelfContained.GrafanaVersion = ""
				}),
			},
			want: "",
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGrafanaVersion(tt.args.indexes, tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaResources_GetGrafanaSubscription(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *v1alpha1.Subscription
	}{
		{
			name: "returns 'grafana-subscription' Subscription",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.ObjectMeta = objectMetaWithNamespace
				}),
			},
			want: &v1alpha1.Subscription{
				ObjectMeta: v12.ObjectMeta{
					Name:      "grafana-subscription",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGrafanaSubscription(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaResources_GetGrafanaOperatorGroup(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *coreosv1.OperatorGroup
	}{
		{
			name: "returns 'observability-operatorgroup' OperatorGroup",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.ObjectMeta = objectMetaWithNamespace
				}),
			},
			want: &coreosv1.OperatorGroup{
				ObjectMeta: v12.ObjectMeta{
					Name:      "observability-operatorgroup",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGrafanaOperatorGroup(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaResources_GetGrafanaProxySecret(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *corev1.Secret
	}{
		{
			name: "returns 'grafana-k8s-proxy' Secret",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.ObjectMeta = objectMetaWithNamespace
				}),
			},
			want: &corev1.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "grafana-k8s-proxy",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGrafanaProxySecret(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaResources_GetGrafanaClusterRole(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *v14.ClusterRole
	}{
		{
			name: "returns Grafana ClusterRole",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &v14.ClusterRole{
				ObjectMeta: v12.ObjectMeta{
					Name: "grafana-oauth-proxy-cluster-role",
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGrafanaClusterRole(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaResources_GetGrafanaClusterRoleBinding(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *v14.ClusterRoleBinding
	}{
		{
			name: "returns Grafana ClusterRoleBinding",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &v14.ClusterRoleBinding{
				ObjectMeta: v12.ObjectMeta{
					Name: "cluster-grafana-oauth-proxy-cluster-role-binding",
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGrafanaClusterRoleBinding(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaResources_GetGrafanaCr(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *v1alpha12.Grafana
	}{
		{
			name: "returns Grafana CR",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &v1alpha12.Grafana{
				ObjectMeta: v12.ObjectMeta{
					Name:      defaultGrafanaName,
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGrafanaCr(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaResources_GetGrafanaDatasource(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *v1alpha12.GrafanaDataSource
	}{
		{
			name: "returns Grafana datasource",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &v1alpha12.GrafanaDataSource{
				ObjectMeta: v12.ObjectMeta{
					Name:      "on-cluster-prometheus",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGrafanaDatasource(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaResources_GetGrafanaDashboardLabelSelectors(t *testing.T) {
	type args struct {
		cr      *v1.Observability
		indexes []v1.RepositoryIndex
	}

	tests := []struct {
		name string
		args args
		want *v12.LabelSelector
	}{
		{
			name: "returns CR GrafanaDashboardLabelSelector when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						GrafanaDashboardLabelSelector: labelSelectorWithNamespace,
					}
				}),
				indexes: []v1.RepositoryIndex{},
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns Grafana dashboard LabelSelector from repo Grafana Index",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: testRepoIndexes,
			},
			want: labelSelectorWithNamespace,
		},
		{
			name: "returns default Grafana dashboard LabelSelector when self contained is nil and no repo config",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: []v1.RepositoryIndex{},
			},
			want: &v12.LabelSelector{
				MatchLabels: defaultGrafanaLabelSelectors,
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGrafanaDashboardLabelSelectors(tt.args.cr, tt.args.indexes)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaResources_GetGrafanaResourceRequirement(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *corev1.ResourceRequirements
	}{
		{
			name: "returns empty ResourceRequirments if NOT self contained",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.ResourceRequirements{},
		},
		{
			name: "returns CR GrafanaResourceRequirement if self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						GrafanaResourceRequirement: &corev1.ResourceRequirements{
							Limits: testResourceList,
						},
					}
				}),
			},
			want: &corev1.ResourceRequirements{
				Limits: testResourceList,
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGrafanaResourceRequirement(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaResources_GetGrafanaOperatorResourceRequirement(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *corev1.ResourceRequirements
	}{
		{
			name: "returns empty ResourceRequirments if NOT self contained",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.ResourceRequirements{},
		},
		{
			name: "returns CR GrafanaOperatorResourceRequirement if self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						GrafanaOperatorResourceRequirement: &corev1.ResourceRequirements{
							Limits: testResourceList,
						},
					}
				}),
			},
			want: &corev1.ResourceRequirements{
				Limits: testResourceList,
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGrafanaOperatorResourceRequirement(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}
