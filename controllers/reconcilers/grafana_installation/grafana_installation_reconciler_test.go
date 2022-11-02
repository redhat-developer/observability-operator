package grafana_installation

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	//"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGrafanaInstallationReconciler_ApproveGrafanaOperatorInstallPlan(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.SchemeBuilder.AddToScheme(scheme)

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
			name: "Success when correct installPlan to be approved is found",
			args: args{
				ctx: context.TODO(),
				cr: &v1.Observability{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
					},
				},
			},
			fields: fields{
				client: fakeclient.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&v1alpha1.InstallPlanList{
					Items: []v1alpha1.InstallPlan{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-install-plan",
								Namespace: "test-namespace",
							},
							Spec: v1alpha1.InstallPlanSpec{
								ClusterServiceVersionNames: []string{GrafanaOperatorDefaultVersion},
								Approved:                   false,
							},
						},
					},
				}).Build(),
			},
			wantErr: false,
		},
		{
			name: "Error when installPlan Update fails",
			args: args{
				ctx: context.TODO(),
				cr: &v1.Observability{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
					},
				},
			},
			fields: fields{
				client: fakeclient.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&v1alpha1.InstallPlanList{
					Items: []v1alpha1.InstallPlan{
						{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "test-namespace",
							},
							Spec: v1alpha1.InstallPlanSpec{
								ClusterServiceVersionNames: []string{GrafanaOperatorDefaultVersion},
								Approved:                   false,
							},
						},
					},
				}).Build(),
			},
			wantErr: true,
		},
		{
			name: "No error when installPlan absent",
			args: args{
				ctx: context.TODO(),
				cr: &v1.Observability{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
					},
				},
			},
			fields: fields{
				client: fakeclient.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
					&v1alpha1.InstallPlanList{
						Items: []v1alpha1.InstallPlan{},
					},
				).Build(),
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

			err := r.approveGrafanaOperatorInstallPlan(test.args.ctx, test.args.cr)
			g.Expect(err != nil).To(Equal(test.wantErr))
		})
	}
}
