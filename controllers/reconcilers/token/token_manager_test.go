package token

import (
	"testing"

	. "github.com/onsi/gomega"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
)

func TestTokenManager_GetObservatoriumTokenSecretName(t *testing.T) {
	g := NewWithT(t)
	config := &v1.ObservatoriumIndex{
		Id: "test-id",
	}

	result := GetObservatoriumTokenSecretName(config)
	g.Expect(result).To(Equal("obs-token-test-id"))
}

func TestTokenManager_GetObservatoriumPrometheusSecretName(t *testing.T) {
	g := NewWithT(t)
	index := &v1.RepositoryIndex{
		Id: "test-id",
		Config: &v1.RepositoryConfig{
			Prometheus: &v1.PrometheusIndex{
				Observatorium: "test-observatorium",
			},
			Observatoria: []v1.ObservatoriumIndex{
				{
					Id: "test-observatorium",
				},
			},
		},
	}

	result := GetObservatoriumPrometheusSecretName(index)
	g.Expect(result).To(Equal("obs-token-test-observatorium"))
}

func TestTokenManager_GetObservatoriumPromtailSecretName(t *testing.T) {
	g := NewWithT(t)
	index := &v1.RepositoryIndex{
		Id: "test-id",
		Config: &v1.RepositoryConfig{
			Promtail: &v1.PromtailIndex{
				Observatorium: "test-observatorium",
			},
			Observatoria: []v1.ObservatoriumIndex{
				{
					Id: "test-observatorium",
				},
			},
		},
	}

	result := GetObservatoriumPromtailSecretName(index)
	g.Expect(result).To(Equal("obs-token-test-observatorium"))
}

func TestTokenManager_GetObservatoriumConfig(t *testing.T) {
	type args struct {
		index *v1.RepositoryIndex
		id    string
	}

	tests := []struct {
		name string
		args args
		want *v1.ObservatoriumIndex
	}{
		{
			name: "returns ObservatoriumIndex with id matching input id",
			args: args{
				index: &v1.RepositoryIndex{
					Id: "test-id",
					Config: &v1.RepositoryConfig{
						Promtail: &v1.PromtailIndex{
							Observatorium: "test-observatorium",
						},
						Observatoria: []v1.ObservatoriumIndex{
							{
								Id: "test-observatorium",
							},
						},
					},
				},
				id: "test-observatorium",
			},
			want: &v1.ObservatoriumIndex{
				Id: "test-observatorium",
			},
		},
		{
			name: "returns nil if repo index is nil",
			args: args{
				index: nil,
				id:    "test-observatorium",
			},
			want: nil,
		},
		{
			name: "returns nil if observatorium id does NOT match input id",
			args: args{
				index: &v1.RepositoryIndex{
					Id: "test-id",
					Config: &v1.RepositoryConfig{
						Promtail: &v1.PromtailIndex{
							Observatorium: "test-observatorium",
						},
						Observatoria: []v1.ObservatoriumIndex{
							{
								Id: "test-observatorium",
							},
						},
					},
				},
				id: "does-not-match",
			},
			want: nil,
		},
	}

	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetObservatoriumConfig(tt.args.index, tt.args.id)
			g.Expect(result).To(Equal(tt.want))
		})
	}

}
