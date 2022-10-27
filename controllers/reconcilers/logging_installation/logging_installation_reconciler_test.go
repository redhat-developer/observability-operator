package logging_installation

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLoggingInstallationReconciler(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.SchemeBuilder.AddToScheme(scheme)
	_ = appsv1.SchemeBuilder.AddToScheme(scheme)

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
		want    v1.ObservabilityStageStatus
	}{
		{
			name: "Success when subscription found",
			args: args{
				ctx: context.TODO(),
				cr: &v1.Observability{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "openshift-logging",
					},
				},
			},
			fields: fields{
				client: fakeclient.NewFakeClientWithScheme(scheme,
					&v1alpha1.Subscription{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cluster-logging",
							Namespace: "openshift-logging",
						},
					},
				),
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
			}

			status, err := r.reconcileSubscription(test.args.ctx, test.args.cr)
			g.Expect(err != nil).To(Equal(test.wantErr))
			g.Expect(status).To(Equal(test.want))
		})
	}
}
