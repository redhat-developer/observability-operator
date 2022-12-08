package utils

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	routev1 "github.com/openshift/api/route/v1"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	ValidVersionString   = "3.1.10"
	InvalidVersionString = "invalid"
)

var (
	testNamespace        = "test-namespace"
	emptyStatefulSetList = &appsv1.StatefulSetList{
		Items: []appsv1.StatefulSet{},
	}
	emptyDeploymentList = &appsv1.DeploymentList{
		Items: []appsv1.Deployment{},
	}
)

func buildObservabilityCR(modifyFn func(obsCR *v1.Observability)) *v1.Observability {
	obsCR := &v1.Observability{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
		},
	}
	if modifyFn != nil {
		modifyFn(obsCR)
	}
	return obsCR
}
func TestReconcilerUtils_IsRouteReady(t *testing.T) {
	type args struct {
		route *routev1.Route
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "return false if route is nil",
			args: args{
				route: nil,
			},
		},
		{
			name: "return false if admitted route condition status is false",
			args: args{
				route: &routev1.Route{
					Status: routev1.RouteStatus{
						Ingress: []routev1.RouteIngress{
							{
								Conditions: []routev1.RouteIngressCondition{
									{
										Type:   routev1.RouteAdmitted,
										Status: corev1.ConditionFalse,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "return false if admitted route condition status is unknown",
			args: args{
				route: &routev1.Route{
					Status: routev1.RouteStatus{
						Ingress: []routev1.RouteIngress{
							{
								Conditions: []routev1.RouteIngressCondition{
									{
										Type:   routev1.RouteAdmitted,
										Status: corev1.ConditionUnknown,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "return true if admitted route condition status is true",
			args: args{
				route: &routev1.Route{
					Status: routev1.RouteStatus{
						Ingress: []routev1.RouteIngress{
							{
								Conditions: []routev1.RouteIngressCondition{
									{
										Type:   routev1.RouteAdmitted,
										Status: corev1.ConditionTrue,
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)
			result := IsRouteReady(test.args.route)
			g.Expect(result).To(Equal(test.want))
		})
	}
}

func TestReconcilerUtils_WaitForGrafanaToBeRemoved(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.SchemeBuilder.AddToScheme(scheme)
	_ = v1.SchemeBuilder.AddToScheme(scheme)
	grafanaDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-deployment",
			Namespace: "test-namespace",
		},
	}
	grafanaDeploymentList := &appsv1.DeploymentList{
		Items: []appsv1.Deployment{*grafanaDeployment},
	}

	type args struct {
		ctx        context.Context
		cr         *v1.Observability
		fakeClient k8sclient.Client
	}

	tests := []struct {
		name    string
		args    args
		want    v1.ObservabilityStageStatus
		wantErr bool
	}{
		{
			name: "return in progress status when Grafana deployment present",
			args: args{
				ctx:        context.TODO(),
				cr:         buildObservabilityCR(nil),
				fakeClient: fakeclient.NewFakeClientWithScheme(scheme, grafanaDeploymentList),
			},
			want:    v1.ResultInProgress,
			wantErr: false,
		},
		{
			name: "return success status when Grafana deployment NOT present",
			args: args{
				ctx:        context.TODO(),
				cr:         buildObservabilityCR(nil),
				fakeClient: fakeclient.NewFakeClientWithScheme(scheme, emptyDeploymentList),
			},
			want:    v1.ResultSuccess,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)
			status, err := WaitForGrafanaToBeRemoved(test.args.ctx, test.args.cr, test.args.fakeClient)
			g.Expect(err != nil).To(Equal(test.wantErr))
			g.Expect(status).To(Equal(test.want))
		})
	}
}

func TestReconcilerUtils_WaitForAlertmanagerToBeRemoved(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.SchemeBuilder.AddToScheme(scheme)
	_ = v1.SchemeBuilder.AddToScheme(scheme)
	alertmanagerStatefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "alertmanager-obs-alertmanager",
			Namespace: "test-namespace",
		},
	}
	alertmanagerStatefulSetList := &appsv1.StatefulSetList{
		Items: []appsv1.StatefulSet{*alertmanagerStatefulSet},
	}

	type args struct {
		ctx        context.Context
		cr         *v1.Observability
		fakeClient k8sclient.Client
	}

	tests := []struct {
		name    string
		args    args
		want    v1.ObservabilityStageStatus
		wantErr bool
	}{
		{
			name: "return in progress status when Alertmanager StatefulSet present",
			args: args{
				ctx:        context.TODO(),
				cr:         buildObservabilityCR(nil),
				fakeClient: fakeclient.NewFakeClientWithScheme(scheme, alertmanagerStatefulSetList),
			},
			want:    v1.ResultInProgress,
			wantErr: false,
		},
		{
			name: "return success status when Alertmanager StatefulSet NOT present",
			args: args{
				ctx:        context.TODO(),
				cr:         buildObservabilityCR(nil),
				fakeClient: fakeclient.NewFakeClientWithScheme(scheme, emptyStatefulSetList),
			},
			want:    v1.ResultSuccess,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)
			status, err := WaitForAlertmanagerToBeRemoved(test.args.ctx, test.args.cr, test.args.fakeClient)
			g.Expect(err != nil).To(Equal(test.wantErr))
			g.Expect(status).To(Equal(test.want))
		})
	}
}

func TestReconcilerUtils_WaitForPrometheusToBeRemoved(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.SchemeBuilder.AddToScheme(scheme)
	_ = v1.SchemeBuilder.AddToScheme(scheme)
	prometheusStatefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-obs-prometheus",
			Namespace: "test-namespace",
		},
	}
	prometheusStatefulSetList := &appsv1.StatefulSetList{
		Items: []appsv1.StatefulSet{*prometheusStatefulSet},
	}

	type args struct {
		ctx        context.Context
		cr         *v1.Observability
		fakeClient k8sclient.Client
	}

	tests := []struct {
		name    string
		args    args
		want    v1.ObservabilityStageStatus
		wantErr bool
	}{
		{
			name: "return in progress status when Prometheus StatefulSet present",
			args: args{
				ctx:        context.TODO(),
				cr:         buildObservabilityCR(nil),
				fakeClient: fakeclient.NewFakeClientWithScheme(scheme, prometheusStatefulSetList),
			},
			want:    v1.ResultInProgress,
			wantErr: false,
		},
		{
			name: "return success status when Prometheus StatefulSet NOT present",
			args: args{
				ctx:        context.TODO(),
				cr:         buildObservabilityCR(nil),
				fakeClient: fakeclient.NewFakeClientWithScheme(scheme, emptyStatefulSetList),
			},
			want:    v1.ResultSuccess,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)
			status, err := WaitForPrometheusToBeRemoved(test.args.ctx, test.args.cr, test.args.fakeClient)
			g.Expect(err != nil).To(Equal(test.wantErr))
			g.Expect(status).To(Equal(test.want))
		})
	}
}
