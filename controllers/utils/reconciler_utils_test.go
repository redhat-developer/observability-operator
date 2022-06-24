package utils

import (
	"testing"

	. "github.com/onsi/gomega"
	routev1 "github.com/openshift/api/route/v1"
	v1 "k8s.io/api/core/v1"
)

const (
	ValidVersionString   = "3.1.10"
	InvalidVersionString = "invalid"
)

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
										Status: v1.ConditionFalse,
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
										Status: v1.ConditionUnknown,
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
										Status: v1.ConditionTrue,
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

	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRouteReady(tt.args.route)
			g.Expect(result).To(Equal(tt.want))
		})
	}
}
