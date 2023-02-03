package model

import (
	"testing"

	. "github.com/onsi/gomega"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigurationResources_GetResourcesService(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *corev1.Service
	}{
		{
			name: "return default resources service",
			args: args{
				cr: &v1.Observability{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
					},
				},
			},
			want: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obs-resources",
					Namespace: "test-namespace",
				},
			},
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)

			service := GetResourcesService(test.args.cr)
			g.Expect(service).To(Equal(test.want))
		})
	}
}

func TestConfigurationResources_GetResourcesDeployment(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *appsv1.Deployment
	}{
		{
			name: "return default deployment",
			args: args{
				cr: &v1.Observability{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
					},
				},
			},
			want: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obs-resources",
					Namespace: "test-namespace",
				},
			},
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)

			deployment := GetResourcesDeployment(test.args.cr)
			g.Expect(deployment).To(Equal(test.want))
		})
	}
}
