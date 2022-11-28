package configuration

import (
	"testing"

	"github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	. "github.com/onsi/gomega"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	dashboardArray  = []string{"test-dashboard-name"}
	dashboardArray2 = []string{"test-dashboard-name-2"}

	testRepoIndexes = []v1.RepositoryIndex{
		{
			Config: nil,
		},
		{
			Config: &v1.RepositoryConfig{
				Grafana: &v1.GrafanaIndex{
					Dashboards: dashboardArray,
				},
				Prometheus: &v1.PrometheusIndex{
					PodMonitors: []string{"pod-monitor-1", "pod-monitor-2"},
					Rules:       []string{"rule-1", "rule-2"},
				},
			},
			BaseUrl:     "test-base-url",
			AccessToken: "test-access-token",
			Tag:         "test-tag",
			Id:          "test-id",
		},
		{
			Config: &v1.RepositoryConfig{
				Prometheus: &v1.PrometheusIndex{
					PodMonitors: []string{"pod-monitor-1", "pod-monitor-2"},
				},
			},
		},
	}
	emptyRepoIndexes = []v1.RepositoryIndex{}
	testRepoIndexes2 = []v1.RepositoryIndex{
		{
			Config: nil,
		},
		{
			Config: &v1.RepositoryConfig{
				Grafana: &v1.GrafanaIndex{
					Dashboards: dashboardArray,
				},
			},
			BaseUrl:     "test-base-url",
			AccessToken: "test-access-token",
			Tag:         "test-tag",
		},
		{
			Config: &v1.RepositoryConfig{
				Grafana: &v1.GrafanaIndex{
					Dashboards: dashboardArray2,
				},
			},
			BaseUrl:     "test-base-url",
			AccessToken: "test-access-token-2",
			Tag:         "test-tag-2",
		},
	}
	testRepoIndexes3 = []v1.RepositoryIndex{
		{
			Config: &v1.RepositoryConfig{
				Grafana: &v1.GrafanaIndex{
					Dashboards: dashboardArray,
				},
			},
			BaseUrl:     "test-base-url",
			AccessToken: "test-access-token",
			Tag:         "test-tag",
		},
		{
			Config: &v1.RepositoryConfig{
				Grafana: &v1.GrafanaIndex{
					Dashboards: dashboardArray,
				},
			},
			BaseUrl:     "test-base-url",
			AccessToken: "test-access-token-2",
			Tag:         "test-tag-2",
		},
	}
	dashboardYaml = []byte(`metadata:
  name: simple-dashboard
spec:
`,
	)
	dashboardYamlWithErrors = []byte(`invalid`)
	dashboardJson           = []byte("test json")
)

func buildObservabilityCR(modifyFn func(obsCR *v1.Observability)) *v1.Observability {
	obsCR := &v1.Observability{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
		},
		Spec: v1.ObservabilitySpec{
			SelfContained: nil,
		},
	}
	if modifyFn != nil {
		modifyFn(obsCR)
	}
	return obsCR
}

func TestGrafanaDashboards_GetNameFromUrl(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get name 'test' from url",
			args: args{
				path: "this/is/a/test",
			},
			want: "test",
		},
		{
			name: "get name 'anotherTest' from url",
			args: args{
				path: "this/is/anotherTest.this.is.excluded",
			},
			want: "anotherTest",
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNameFromUrl(tt.args.path)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaDashboards_GetUniqueDashboards(t *testing.T) {
	type args struct {
		indexes []v1.RepositoryIndex
	}
	tests := []struct {
		name string
		args args
		want []DashboardInfo
	}{
		{
			name: "success getting DashboardInfo",
			args: args{
				indexes: testRepoIndexes,
			},
			want: []DashboardInfo{
				{
					Name:        "test-dashboard-name",
					Url:         "test-base-url/test-dashboard-name",
					AccessToken: "test-access-token",
					Tag:         "test-tag",
				},
			},
		},
		{
			name: "empty repo",
			args: args{
				indexes: emptyRepoIndexes,
			},
			want: nil,
		},
		{
			name: "success getting multiple DashboardInfos with unique names",
			args: args{
				indexes: testRepoIndexes2,
			},
			want: []DashboardInfo{
				{
					Name:        "test-dashboard-name",
					Url:         "test-base-url/test-dashboard-name",
					AccessToken: "test-access-token",
					Tag:         "test-tag",
				},
				{
					Name:        "test-dashboard-name-2",
					Url:         "test-base-url/test-dashboard-name-2",
					AccessToken: "test-access-token-2",
					Tag:         "test-tag-2",
				},
			},
		},
		{
			name: "success NOT returning DashboardInfos with duplicate name",
			args: args{
				indexes: testRepoIndexes3,
			},
			want: []DashboardInfo{
				{
					Name:        "test-dashboard-name",
					Url:         "test-base-url/test-dashboard-name",
					AccessToken: "test-access-token",
					Tag:         "test-tag",
				},
			},
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUniqueDashboards(tt.args.indexes)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaDashboards_ParseDashboardFromYaml(t *testing.T) {
	type args struct {
		cr     *v1.Observability
		name   string
		source []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *v1alpha1.GrafanaDashboard
		wantErr bool
	}{
		{
			name: "success parsing dashboard from yaml",
			args: args{
				cr:     buildObservabilityCR(nil),
				name:   "test-dashboard-name",
				source: dashboardYaml,
			},
			want: &v1alpha1.GrafanaDashboard{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-dashboard-name",
				},
			},
		},
		{
			name: "error if yaml is invalid",
			args: args{
				cr:     buildObservabilityCR(nil),
				name:   "test-dashboard-name",
				source: dashboardYamlWithErrors,
			},
			want:    nil,
			wantErr: true,
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDashboardFromYaml(tt.args.cr, tt.args.name, tt.args.source)
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaDashboards_CreateDashboardFromSource(t *testing.T) {
	type args struct {
		cr     *v1.Observability
		name   string
		t      SourceType
		source []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *v1alpha1.GrafanaDashboard
		wantErr bool
	}{
		{
			name: "success creating dashboard from JSON source",
			args: args{
				cr:     buildObservabilityCR(nil),
				name:   "test-dashboard-name",
				t:      1,
				source: dashboardJson,
			},
			want: &v1alpha1.GrafanaDashboard{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-dashboard-name",
				},
				Spec: v1alpha1.GrafanaDashboardSpec{
					Json: "test json",
				},
			},
		},
		{
			name: "success creating dashboard from Jsonnet source",
			args: args{
				cr:     buildObservabilityCR(nil),
				name:   "test-dashboard-name",
				t:      2,
				source: dashboardJson,
			},
			want: &v1alpha1.GrafanaDashboard{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-dashboard-name",
				},
				Spec: v1alpha1.GrafanaDashboardSpec{
					Jsonnet: "test json",
				},
			},
		},
		{
			name: "error creating dashboard with other source type",
			args: args{
				cr:     buildObservabilityCR(nil),
				name:   "test-dashboard-name",
				t:      3,
				source: dashboardJson,
			},
			want:    nil,
			wantErr: true,
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := createDashboardFromSource(tt.args.cr, tt.args.name, tt.args.t, tt.args.source)
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGrafanaDashboards_GetFileType(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want SourceType
	}{
		{
			name: "returns 4 when path is empty",
			args: args{
				path: "",
			},
			want: 4,
		},
		{
			name: "returns 1 when path includes json extension",
			args: args{
				path: "path/to/type.json",
			},
			want: 1,
		},
		{
			name: "returns 2 when path includes grafonnet extension",
			args: args{
				path: "path/to/type.grafonnet",
			},
			want: 2,
		},
		{
			name: "returns 2 when path includes jsonnet extension",
			args: args{
				path: "path/to/type.jsonnet",
			},
			want: 2,
		},
		{
			name: "returns 3 when path includes yaml extension",
			args: args{
				path: "path/to/type.yaml",
			},
			want: 3,
		},
		{
			name: "returns 4 when path includes unknown extension",
			args: args{
				path: "path/to/type.unknown",
			},
			want: 4,
		},
		{
			name: "returns 4 when path has no extension",
			args: args{
				path: "path/to/type",
			},
			want: 4,
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFileType(tt.args.path)
			Expect(result).To(Equal(tt.want))
		})
	}
}
