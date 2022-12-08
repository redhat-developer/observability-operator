package model

import (
	"testing"

	. "github.com/onsi/gomega"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTokenResources_GetTokenSecret(t *testing.T) {
	type args struct {
		cr   *v1.Observability
		name string
	}
	tests := []struct {
		name string
		args args
		want *corev1.Secret
	}{
		{
			name: "return cr AlertManagerDefaultName if self contained",
			args: args{
				cr:   buildObservabilityCR(nil),
				name: "test-secret-name",
			},
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret-name",
					Namespace: testNamespace,
					Labels: map[string]string{
						"managed-by": "observability-operator",
						"purpose":    "observatorium-token-secret",
					},
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTokenSecret(tt.args.cr, tt.args.name)
			Expect(result).To(Equal(tt.want))
		})
	}
}
