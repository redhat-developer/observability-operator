package configuration

import (
	"testing"

	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
)

func TestGetOriginOauthProxyImage(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test default image is returned when self contained is nil in cr",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: DefaultOriginOauthProxyImage,
		},
		{
			name: "test default image is returned when empty in spec",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						OriginOauthProxyImage: "",
					}
				}),
			},
			want: DefaultOriginOauthProxyImage,
		},
		{
			name: "test image in spec is returned when not empty",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						OriginOauthProxyImage: "customOriginOauthProxyImage",
					}
				}),
			},
			want: "customOriginOauthProxyImage",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOriginOauthProxyImage(tt.args.cr); got != tt.want {
				t.Errorf("GetOriginOauthProxyImage() = %v, want %v", got, tt.want)
			}
		})
	}
}
