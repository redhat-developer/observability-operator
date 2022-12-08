package model

import (
	"testing"

	. "github.com/onsi/gomega"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v14 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTokenRefresherResources_GetTokenRefresherName(t *testing.T) {
	type args struct {
		id string
		t  TokenRefresherType
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "return default token refresher name",
			args: args{
				id: "test-id",
				t:  "test-token-refresher-type",
			},
			want: "token-refresher-test-token-refresher-type-test-id",
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTokenRefresherName(tt.args.id, tt.args.t)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestTokenRefresherResources_GetTokenRefresherService(t *testing.T) {
	type args struct {
		cr   *v1.Observability
		name string
	}
	tests := []struct {
		name string
		args args
		want *corev1.Service
	}{
		{
			name: "return token refresher service",
			args: args{
				cr:   buildObservabilityCR(nil),
				name: "test-service-name",
			},
			want: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service-name",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTokenRefresherService(tt.args.cr, tt.args.name)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestTokenRefresherResources_GetTokenRefresherDeployment(t *testing.T) {
	type args struct {
		cr   *v1.Observability
		name string
	}
	tests := []struct {
		name string
		args args
		want *appsv1.Deployment
	}{
		{
			name: "return token refresher deployment",
			args: args{
				cr:   buildObservabilityCR(nil),
				name: "test-deployment-name",
			},
			want: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment-name",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTokenRefresherDeployment(tt.args.cr, tt.args.name)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestTokenRefresherResources_GetTokenRefresherNetworkPolicy(t *testing.T) {
	type args struct {
		cr   *v1.Observability
		name string
	}
	tests := []struct {
		name string
		args args
		want *v14.NetworkPolicy
	}{
		{
			name: "return token refresher network policy",
			args: args{
				cr:   buildObservabilityCR(nil),
				name: "test-name",
			},
			want: &v14.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-name-network-policy",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTokenRefresherNetworkPolicy(tt.args.cr, tt.args.name)
			Expect(result).To(Equal(tt.want))
		})
	}
}
