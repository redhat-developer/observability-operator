package model

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"testing"
)

func TestGetPrometheusStorageSize(t *testing.T) {
	type args struct {
		cr      *v1.Observability
		indexes []v1.RepositoryIndex
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "cr storage is used when selfcontained is specified AND a storage value is provided",
			args: args{
				cr: &v1.Observability{
					Spec: v1.ObservabilitySpec{
						Storage: &v1.Storage{
							PrometheusStorageSpec: &monitoringv1.StorageSpec{
								VolumeClaimTemplate: monitoringv1.EmbeddedPersistentVolumeClaim{
									Spec: corev1.PersistentVolumeClaimSpec{
										Resources: corev1.ResourceRequirements{
											Requests: map[corev1.ResourceName]resource.Quantity{
												corev1.ResourceStorage: resource.MustParse("10Gi"),
											},
										},
									},
								},
							},
						},
						SelfContained: &v1.SelfContained{},
					},
				},
			},
			want: "10Gi",
		},
		{
			name: "default storage is used when selfcontained is NOT specified AND NO storage value is provided",
			args: args{
				cr: &v1.Observability{
					Spec: v1.ObservabilitySpec{},
				},
			},
			want: "250Gi",
		},
		{
			name: "no nil failure when selfcontained is specified AND a storage value is NOT provided",
			args: args{
				cr: &v1.Observability{
					Spec: v1.ObservabilitySpec{
						SelfContained: &v1.SelfContained{},
					},
				},
			},
			want: "250Gi",
		},
		{
			name: "no nil failure when selfcontained is specified AND PersistentVolumeClaim is NOT provided",
			args: args{
				cr: &v1.Observability{
					Spec: v1.ObservabilitySpec{
						SelfContained: &v1.SelfContained{},
						Storage: &v1.Storage{
							PrometheusStorageSpec: &monitoringv1.StorageSpec{
								VolumeClaimTemplate: monitoringv1.EmbeddedPersistentVolumeClaim{
									Spec: corev1.PersistentVolumeClaimSpec{},
								},
							},
						},
					},
				},
			},
			want: "250Gi",
		},
		{
			name: "no nil failure when selfcontained is specified AND EmbeddedPersistentVolumeClaim is NOT provided",
			args: args{
				cr: &v1.Observability{
					Spec: v1.ObservabilitySpec{
						SelfContained: &v1.SelfContained{},
						Storage: &v1.Storage{
							PrometheusStorageSpec: &monitoringv1.StorageSpec{},
						},
					},
				},
			},
			want: "250Gi",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPrometheusStorageSize(tt.args.cr, tt.args.indexes); got != tt.want {
				t.Errorf("GetPrometheusStorageSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
