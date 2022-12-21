package configuration

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestObservabilityResources_ReconcileResourcesDeployment(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.SchemeBuilder.AddToScheme(scheme)

	type fields struct {
		client client.Client
	}

	type args struct {
		ctx   context.Context
		cr    *v1.Observability
		image string
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "no error when resources deployment created",
			args: args{
				ctx: context.Background(),
				cr: &v1.Observability{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
					},
				},
				image: "test-image",
			},
			fields: fields{
				client: fakeclient.NewFakeClientWithScheme(scheme),
			},
			wantErr: false,
		},
		{
			name: "no error if resources deployment updated",
			args: args{
				ctx: context.Background(),
				cr: &v1.Observability{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
					},
				},
				image: "test-image",
			},
			fields: fields{
				client: fakeclient.NewFakeClientWithScheme(scheme,
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "obs-resources",
							Namespace: "test-namespace",
						},
					},
				),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)
			r := &Reconciler{
				client: test.fields.client,
			}

			err := r.ReconcileResourcesDeployment(test.args.ctx, test.args.cr, test.args.image)
			g.Expect(err != nil).To(Equal(test.wantErr))
		})
	}
}

func TestObservabilityResources_ReconcileResourcesService(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.SchemeBuilder.AddToScheme(scheme)

	type fields struct {
		client client.Client
	}

	type args struct {
		ctx context.Context
		cr  *v1.Observability
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "no error when resources deployment created",
			args: args{
				ctx: context.Background(),
				cr: &v1.Observability{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
					},
				},
			},
			fields: fields{
				client: fakeclient.NewFakeClientWithScheme(scheme),
			},
			wantErr: false,
		},
		{
			name: "no error if resources deployment updated",
			args: args{
				ctx: context.Background(),
				cr: &v1.Observability{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
					},
				},
			},
			fields: fields{
				client: fakeclient.NewFakeClientWithScheme(scheme,
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "obs-resources",
							Namespace: "test-namespace",
						},
					},
				),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)
			r := &Reconciler{
				client: test.fields.client,
			}

			err := r.ReconcileResourcesService(test.args.ctx, test.args.cr)
			g.Expect(err != nil).To(Equal(test.wantErr))
		})
	}
}
