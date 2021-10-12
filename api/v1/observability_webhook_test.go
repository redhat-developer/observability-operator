package v1

import (
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestObservability_ValidateUpdate(t *testing.T) {
	type fields struct {
		TypeMeta   v12.TypeMeta
		ObjectMeta v12.ObjectMeta
		Spec       ObservabilitySpec
		Status     ObservabilityStatus
	}
	type args struct {
		old runtime.Object
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "AlertManagerDefaultName - error if it's old is nil and new got set",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{},
				},
			},
			args: args{old: &Observability{
				Spec: ObservabilitySpec{
					AlertManagerDefaultName: "something",
				},
			}},
			wantErr: true,
		},
		{
			name: "AlertManagerDefaultName - error if changed",
			fields: fields{
				Spec: ObservabilitySpec{
					AlertManagerDefaultName: "something",
				},
			},
			args: args{old: &Observability{
				Spec: ObservabilitySpec{
					AlertManagerDefaultName: "somethingelse",
				},
			}},
			wantErr: true,
		},
		{
			name: "AlertManagerDefaultName - no error if not changed",
			fields: fields{
				Spec: ObservabilitySpec{
					AlertManagerDefaultName: "something",
				},
			},
			args: args{old: &Observability{
				Spec: ObservabilitySpec{
					AlertManagerDefaultName: "something",
				},
			}},
			wantErr: false,
		},
		{
			name: "AlertManagerDefaultName -  no error on nil alertmanager objects",
			fields: fields{
				Spec: ObservabilitySpec{},
			},
			args: args{old: &Observability{
				Spec: ObservabilitySpec{},
			}},
			wantErr: false,
		},
		{
			name: "PrometheusDefaultName - error if it's old is nil and new got set",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{},
				},
			},
			args: args{old: &Observability{
				Spec: ObservabilitySpec{
					PrometheusDefaultName: "something",
				},
			}},
			wantErr: true,
		},
		{
			name: "PrometheusDefaultName - error if changed",
			fields: fields{
				Spec: ObservabilitySpec{
					PrometheusDefaultName: "something",
				},
			},
			args: args{old: &Observability{
				Spec: ObservabilitySpec{
					PrometheusDefaultName: "somethingelse",
				},
			}},
			wantErr: true,
		},
		{
			name: "PrometheusDefaultName - no error if not changed",
			fields: fields{
				Spec: ObservabilitySpec{
					PrometheusDefaultName: "something",
				},
			},
			args: args{old: &Observability{
				Spec: ObservabilitySpec{
					PrometheusDefaultName: "something",
				},
			}},
			wantErr: false,
		},
		{
			name: "PrometheusDefaultName -  no error on nil prometheus objects",
			fields: fields{
				Spec: ObservabilitySpec{
					AlertManagerDefaultName: "",
				},
			},
			args: args{old: &Observability{
				Spec: ObservabilitySpec{
					AlertManagerDefaultName: "",
				},
			}},
			wantErr: false,
		},
		{
			name: "GrafanaDefaultName - error if it's old is nil and new got set",
			fields: fields{
				Spec: ObservabilitySpec{},
			},
			args: args{old: &Observability{
				Spec: ObservabilitySpec{
					GrafanaDefaultName: "something",
				},
			}},
			wantErr: true,
		},
		{
			name: "GrafanaDefaultName - error if changed",
			fields: fields{
				Spec: ObservabilitySpec{
					GrafanaDefaultName: "something",
				},
			},
			args: args{old: &Observability{
				Spec: ObservabilitySpec{
					GrafanaDefaultName: "somethingelse",
				},
			}},
			wantErr: true,
		},
		{
			name: "GrafanaDefaultName - no error if not changed",
			fields: fields{
				Spec: ObservabilitySpec{
					GrafanaDefaultName: "something",
				},
			},
			args: args{old: &Observability{
				Spec: ObservabilitySpec{
					GrafanaDefaultName: "something",
				},
			}},
			wantErr: false,
		},
		{
			name: "GrafanaDefaultName -  no error on nil prometheus objects",
			fields: fields{
				Spec: ObservabilitySpec{
					AlertManagerDefaultName: "",
					PrometheusDefaultName:   "",
				},
			},
			args: args{old: &Observability{
				Spec: ObservabilitySpec{
					AlertManagerDefaultName: "",
					PrometheusDefaultName:   "",
				},
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := &Observability{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if err := in.ValidateUpdate(tt.args.old); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
