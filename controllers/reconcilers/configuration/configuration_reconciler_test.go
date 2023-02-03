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

func TestConfigurationReconciler_WaitForResourcesService(t *testing.T) {
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
		want    v1.ObservabilityStageStatus
	}{
		{
			name: "Success when resources service ready",
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
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "obs-resources",
							Namespace: "test-namespace",
						},
						Status: corev1.ServiceStatus{
							Conditions: []metav1.Condition{
								{
									Type:   "Ready",
									Status: "True",
								},
							},
						},
					},
				),
			},
			wantErr: false,
			want:    v1.ResultSuccess,
		},
		{
			name: "In progress if resources service not found",
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
					&corev1.Service{},
				),
			},
			wantErr: false,
			want:    v1.ResultInProgress,
		},
		{
			name: "In progress if resources service not ready",
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
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "obs-resources",
							Namespace: "test-namespace",
						},
						Status: corev1.ServiceStatus{
							Conditions: []metav1.Condition{
								{
									Type:   "Ready",
									Status: "False",
								},
							},
						},
					},
				),
			},
			wantErr: false,
			want:    v1.ResultInProgress,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)
			r := &Reconciler{
				client: test.fields.client,
			}

			status, err := r.waitForResourcesService(test.args.ctx, test.args.cr)
			g.Expect(err != nil).To(Equal(test.wantErr))
			g.Expect(status).To(Equal(test.want))
		})
	}
}

func TestConfigurationReconciler_WaitForResourcesDeployment(t *testing.T) {
	scheme := runtime.NewScheme()
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
			name: "Success when resources deployment ready",
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
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "obs-resources",
							Namespace: "test-namespace",
						},
						Status: appsv1.DeploymentStatus{
							ReadyReplicas: 1,
						},
					},
				),
			},
			wantErr: false,
			want:    v1.ResultSuccess,
		},
		{
			name: "In progress if resources deployment not ready",
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
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "obs-resources",
							Namespace: "test-namespace",
						},
						Status: appsv1.DeploymentStatus{
							ReadyReplicas: 0,
						},
					},
				),
			},
			wantErr: false,
			want:    v1.ResultInProgress,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)
			r := &Reconciler{
				client: test.fields.client,
			}

			status, err := r.waitForResourcesDeployment(test.args.ctx, test.args.cr)
			g.Expect(err != nil).To(Equal(test.wantErr))
			g.Expect(status).To(Equal(test.want))
		})
	}
}
