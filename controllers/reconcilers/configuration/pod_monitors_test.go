package configuration

import (
	"testing"

	. "github.com/onsi/gomega"
	v12 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	podMonitorYaml = []byte(`metadata:
  name: test-name
  spec:
  `)
	invalidPodMonitorYaml = []byte(`invalid`)
	requestedLabels       = map[string]string{"requested-label-1": "requested-test-1", "requested-label-2": "requested-test-2"}
	existingLabels        = map[string]string{"existing-label-1": "existing-test-1", "existing-label-2": "existing-test-2"}
)

func TestPodMonitors_MergeLabels(t *testing.T) {
	type args struct {
		requested map[string]string
		existing  map[string]string
	}

	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "returns requested if existing is nil",
			args: args{
				requested: requestedLabels,
				existing:  nil,
			},
			want: map[string]string{"requested-label-1": "requested-test-1", "requested-label-2": "requested-test-2"},
		},
		{
			name: "success merging labels",
			args: args{
				requested: requestedLabels,
				existing:  existingLabels,
			},
			want: map[string]string{"existing-label-1": "existing-test-1", "existing-label-2": "existing-test-2", "requested-label-1": "requested-test-1", "requested-label-2": "requested-test-2"},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeLabels(tt.args.requested, tt.args.existing)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPodMonitors_GetUniquePodMonitors(t *testing.T) {
	type args struct {
		indexes []v1.RepositoryIndex
	}
	tests := []struct {
		name string
		args args
		want []ResourceInfo
	}{
		{
			name: "success returning ResourceInfo from repo",
			args: args{
				indexes: testRepoIndexes,
			},
			want: []ResourceInfo{
				{
					Id:          "test-id",
					Name:        "pod-monitor-1",
					Url:         "test-base-url/pod-monitor-1",
					AccessToken: "test-access-token",
					Tag:         "test-tag",
				},
				{
					Id:          "test-id",
					Name:        "pod-monitor-2",
					Url:         "test-base-url/pod-monitor-2",
					AccessToken: "test-access-token",
					Tag:         "test-tag",
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUniquePodMonitors(tt.args.indexes)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPodMonitors_ParsePodMonitorFromYaml(t *testing.T) {
	type args struct {
		cr     *v1.Observability
		name   string
		source []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *v12.PodMonitor
		wantErr bool
	}{
		{
			name: "success parsing PodMonitor from yaml",
			args: args{
				cr:     buildObservabilityCR(nil),
				name:   "test-name",
				source: podMonitorYaml,
			},
			want: &v12.PodMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-name",
					Namespace: "test-namespace",
				},
			},
		},
		{
			name: "error parsing PodMonitor from invalid yaml",
			args: args{
				cr:     buildObservabilityCR(nil),
				name:   "test-name",
				source: invalidPodMonitorYaml,
			},
			want:    nil,
			wantErr: true,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePodMonitorFromYaml(tt.args.cr, tt.args.name, tt.args.source)
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(result).To(Equal(tt.want))
		})
	}
}
