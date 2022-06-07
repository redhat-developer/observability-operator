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

func TestReconcilerUtils_HasNewerOrSameClusterVersion(t *testing.T) {
	type args struct {
		test      string
		compareTo string
	}

	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "error when compareTo is NOT valid version",
			args: args{
				test:      ValidVersionString,
				compareTo: InvalidVersionString,
			},
			wantErr: true,
		},
		{
			name: "error when test is NOT valid version",
			args: args{
				test:      InvalidVersionString,
				compareTo: ValidVersionString,
			},
			wantErr: true,
		},
		{
			name: "return true when test version major is greater than compareTo version major",
			args: args{
				test:      "4.0.0",
				compareTo: ValidVersionString,
			},
			want: true,
		},
		{
			name: "return true when test version minor is greater than or equal to compareTo version minor",
			args: args{
				test:      "3.1.5",
				compareTo: ValidVersionString,
			},
			want: true,
		},
		{
			name: "return false when test version minor is less than compareTo version minor",
			args: args{
				test:      "3.0.5",
				compareTo: ValidVersionString,
			},
		},
	}

	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HasNewerOrSameClusterMinorVersion(tt.args.test, tt.args.compareTo)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(result).To(Equal(tt.want))
		})
	}
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
