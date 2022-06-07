package configuration

import (
	"testing"

	. "github.com/onsi/gomega"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	kv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testRepoInvalidStorageOverride = []v1.RepositoryIndex{
		{
			Config: &v1.RepositoryConfig{
				Prometheus: &v1.PrometheusIndex{
					OverridePrometheusPvcSize: "3ti",
				},
			},
		},
	}
	testRepoValidStorageOverride = []v1.RepositoryIndex{
		{
			Config: &v1.RepositoryConfig{
				Prometheus: &v1.PrometheusIndex{
					OverridePrometheusPvcSize: "3Gi",
				},
			},
		},
	}
	overrideStorageSpec = &prometheusv1.StorageSpec{
		VolumeClaimTemplate: prometheusv1.EmbeddedPersistentVolumeClaim{
			EmbeddedObjectMetadata: prometheusv1.EmbeddedObjectMetadata{
				Name: "managed-services",
			},
			Spec: kv1.PersistentVolumeClaimSpec{
				Resources: kv1.ResourceRequirements{
					Requests: map[kv1.ResourceName]resource.Quantity{
						kv1.ResourceStorage: resource.MustParse("3Gi"),
					},
				},
			},
		},
	}
	crStorageSpec = &prometheusv1.StorageSpec{
		VolumeClaimTemplate: prometheusv1.EmbeddedPersistentVolumeClaim{
			EmbeddedObjectMetadata: prometheusv1.EmbeddedObjectMetadata{
				Name: "managed-services",
			},
			Spec: kv1.PersistentVolumeClaimSpec{
				Resources: kv1.ResourceRequirements{
					Requests: map[kv1.ResourceName]resource.Quantity{
						kv1.ResourceStorage: resource.MustParse("2Gi"),
					},
				},
			},
		},
	}
	defaultStorageSpec = &prometheusv1.StorageSpec{
		VolumeClaimTemplate: prometheusv1.EmbeddedPersistentVolumeClaim{
			EmbeddedObjectMetadata: prometheusv1.EmbeddedObjectMetadata{
				Name: "managed-services",
			},
			Spec: kv1.PersistentVolumeClaimSpec{
				Resources: kv1.ResourceRequirements{
					Requests: map[kv1.ResourceName]resource.Quantity{
						kv1.ResourceStorage: resource.MustParse("250Gi"),
					},
				},
			},
		},
	}
)

func buildObservabilityCRForPrometheus(modifyFn func(obsCR *v1.Observability)) *v1.Observability {
	obsCR := &v1.Observability{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
		},
		Spec: v1.ObservabilitySpec{
			SelfContained: nil,
			Storage: &v1.Storage{
				PrometheusStorageSpec: &prometheusv1.StorageSpec{
					VolumeClaimTemplate: prometheusv1.EmbeddedPersistentVolumeClaim{
						EmbeddedObjectMetadata: prometheusv1.EmbeddedObjectMetadata{
							Name: "managed-services",
						},
						Spec: kv1.PersistentVolumeClaimSpec{
							Resources: kv1.ResourceRequirements{
								Requests: map[kv1.ResourceName]resource.Quantity{kv1.ResourceStorage: resource.MustParse("2Gi")},
							},
						},
					},
				},
			},
		},
	}
	if modifyFn != nil {
		modifyFn(obsCR)
	}
	return obsCR
}

func TestPrometheus_GetPrometheusStorageSpecHelper(t *testing.T) {
	type args struct {
		cr      *v1.Observability
		indexes []v1.RepositoryIndex
	}

	tests := []struct {
		name    string
		args    args
		want    *prometheusv1.StorageSpec
		wantErr bool
	}{
		{
			name: "success when valid override storage size provided in repo",
			args: args{
				cr:      buildObservabilityCRForPrometheus(nil),
				indexes: testRepoValidStorageOverride,
			},
			want: overrideStorageSpec,
		},
		{
			name: "cr storage is returned when cr is self-contained and repo override empty",
			args: args{
				cr: buildObservabilityCRForPrometheus(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
				}),
				indexes: emptyRepoIndexes,
			},
			want: crStorageSpec,
		},
		{
			name: "error when override storage size is not valid and cr storage is returned",
			args: args{
				cr:      buildObservabilityCRForPrometheus(nil),
				indexes: testRepoInvalidStorageOverride,
			},
			want:    crStorageSpec,
			wantErr: true,
		},
		{
			name: "return default 250Gi storage when NOT self contained and override is empty",
			args: args{
				cr:      buildObservabilityCRForPrometheus(nil),
				indexes: emptyRepoIndexes,
			},
			want: defaultStorageSpec,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getPrometheusStorageSpecHelper(tt.args.cr, tt.args.indexes)
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheus_GetRetentionHelper(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "success when valid retention value returned",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.Retention = "30d"
				}),
			},
			want: "30d",
		},
		{
			name: "default returned when CR value is invalid",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.Retention = "30r"
				}),
			},
			want: "45d",
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRetentionHelper(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}
