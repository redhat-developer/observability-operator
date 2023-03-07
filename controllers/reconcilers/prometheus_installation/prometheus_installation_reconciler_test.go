package prometheus_installation

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPrometheusInstallationReconciler_ReconcileCatalogSource(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.SchemeBuilder.AddToScheme(scheme)
	_ = appsv1.SchemeBuilder.AddToScheme(scheme)

	type fields struct {
		client client.Client
		scheme *runtime.Scheme
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
		want    v1.ObservabilityStageStatus
	}{
		{
			name: "Success when catalogsource found with old custom index image and removed",
			args: args{
				ctx: context.TODO(),
				cr: &v1.Observability{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
					},
				},
			},
			fields: fields{
				client: fakeclient.NewFakeClientWithScheme(scheme,
					&v1alpha1.CatalogSource{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "prometheus-catalogsource",
							Namespace: "test-namespace",
						},
						Spec: v1alpha1.CatalogSourceSpec{
							Image: "quay.io/integreatly/custom-prometheus-index:2.0.0",
						},
					},
				),
			},
			wantErr: false,
			want:    v1.ResultSuccess,
		},
		{
			name: "Success when catalogsource with old custom index image not found",
			args: args{
				ctx: context.TODO(),
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
			want:    v1.ResultSuccess,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)
			r := &Reconciler{
				client: test.fields.client,
				scheme: test.fields.scheme,
			}

			status, err := r.reconcileCatalogSource(test.args.ctx, test.args.cr)
			g.Expect(err != nil).To(Equal(test.wantErr))
			g.Expect(status).To(Equal(test.want))
		})
	}
}
