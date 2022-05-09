package v1

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const secretContent = "secret content"

func TestObservabilityTypes_ExternalSyncDisabled(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       ObservabilitySpec
		Status     ObservabilityStatus
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "true if spec is self contained and repo sync disabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						DisableRepoSync: &([]bool{true})[0],
					},
				},
			},
			want: true,
		},
		{
			name: "false if spec is not self contained",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: nil,
				},
			},
			want: false,
		},
		{
			name: "false if spec is self contained and repo sync enabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						DisableRepoSync: &([]bool{false})[0],
					},
				},
			},
			want: false,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &Observability{
				tt.fields.TypeMeta,
				tt.fields.ObjectMeta,
				tt.fields.Spec,
				tt.fields.Status,
			}
			result := obs.ExternalSyncDisabled()
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestObservabilityTypes_OverrideSelectors(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       ObservabilitySpec
		Status     ObservabilityStatus
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "true if spec is self contained and override selectors is enabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						OverrideSelectors: &([]bool{true})[0],
					},
				},
			},
			want: true,
		},
		{
			name: "false if spec is not self contained",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: nil,
				},
			},
			want: false,
		},
		{
			name: "false if spec is self contained and override selectors disabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						OverrideSelectors: &([]bool{false})[0],
					},
				},
			},
			want: false,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &Observability{
				tt.fields.TypeMeta,
				tt.fields.ObjectMeta,
				tt.fields.Spec,
				tt.fields.Status,
			}
			result := obs.OverrideSelectors()
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestObservabilityTypes_ObservatoriumDisabled(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       ObservabilitySpec
		Status     ObservabilityStatus
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "true if spec is self contained and Observatorium disabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						DisableObservatorium: &([]bool{true})[0],
					},
				},
			},
			want: true,
		},
		{
			name: "false if spec is not self contained",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: nil,
				},
			},
			want: false,
		},
		{
			name: "false if spec is self contained and Observatorium disabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						DisableObservatorium: &([]bool{false})[0],
					},
				},
			},
			want: false,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &Observability{
				tt.fields.TypeMeta,
				tt.fields.ObjectMeta,
				tt.fields.Spec,
				tt.fields.Status,
			}
			result := obs.ObservatoriumDisabled()
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestObservabilityTypes_PagerDutyDisabled(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       ObservabilitySpec
		Status     ObservabilityStatus
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "true if spec is self contained and Pager Duty disabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						DisablePagerDuty: &([]bool{true})[0],
					},
				},
			},
			want: true,
		},
		{
			name: "false if spec is not self contained",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: nil,
				},
			},
			want: false,
		},
		{
			name: "false if spec is self contained and Pager Duty enabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						DisablePagerDuty: &([]bool{false})[0],
					},
				},
			},
			want: false,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &Observability{
				tt.fields.TypeMeta,
				tt.fields.ObjectMeta,
				tt.fields.Spec,
				tt.fields.Status,
			}
			result := obs.PagerDutyDisabled()
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestObservabilityTypes_DeadMansSnitchDisabled(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       ObservabilitySpec
		Status     ObservabilityStatus
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "true if spec is self contained and Dead Mans Snitch disabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						DisableDeadmansSnitch: &([]bool{true})[0],
					},
				},
			},
			want: true,
		},
		{
			name: "false if spec is not self contained",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: nil,
				},
			},
			want: false,
		},
		{
			name: "false if spec is self contained and Dead Mans Snitch enabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						DisableDeadmansSnitch: &([]bool{false})[0],
					},
				},
			},
			want: false,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &Observability{
				tt.fields.TypeMeta,
				tt.fields.ObjectMeta,
				tt.fields.Spec,
				tt.fields.Status,
			}
			result := obs.DeadMansSnitchDisabled()
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestObservabilityTypes_SmtpDisabled(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       ObservabilitySpec
		Status     ObservabilityStatus
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "true if spec is self contained and SMTP disabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						DisableSmtp: &([]bool{true})[0],
					},
				},
			},
			want: true,
		},
		{
			name: "false if spec is not self contained",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: nil,
				},
			},
			want: false,
		},
		{
			name: "false if spec is self contained and SMTP enabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						DisableSmtp: &([]bool{false})[0],
					},
				},
			},
			want: false,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &Observability{
				tt.fields.TypeMeta,
				tt.fields.ObjectMeta,
				tt.fields.Spec,
				tt.fields.Status,
			}
			result := obs.SmtpDisabled()
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestObservabilityTypes_BlackboxExporterDisabled(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       ObservabilitySpec
		Status     ObservabilityStatus
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "true if spec is self contained and Blackbox Exporter disabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						DisableBlackboxExporter: &([]bool{true})[0],
					},
				},
			},
			want: true,
		},
		{
			name: "false if spec is not self contained",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: nil,
				},
			},
			want: false,
		},
		{
			name: "false if spec is self contained and Blackbox Exporter enabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						DisableBlackboxExporter: &([]bool{false})[0],
					},
				},
			},
			want: false,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &Observability{
				tt.fields.TypeMeta,
				tt.fields.ObjectMeta,
				tt.fields.Spec,
				tt.fields.Status,
			}
			result := obs.BlackboxExporterDisabled()
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestObservabilityTypes_SelfSignedCerts(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       ObservabilitySpec
		Status     ObservabilityStatus
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "true if spec is self contained and self signed certs enabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						SelfSignedCerts: &([]bool{true})[0],
					},
				},
			},
			want: true,
		},
		{
			name: "false if spec is not self contained",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: nil,
				},
			},
			want: false,
		},
		{
			name: "false if spec is self contained and Blackbox Exporter enabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						SelfSignedCerts: &([]bool{false})[0],
					},
				},
			},
			want: false,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &Observability{
				tt.fields.TypeMeta,
				tt.fields.ObjectMeta,
				tt.fields.Spec,
				tt.fields.Status,
			}
			result := obs.SelfSignedCerts()
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestObservabilityTypes_HasAlertmanagerConfigSecret(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       ObservabilitySpec
		Status     ObservabilityStatus
	}

	type want struct {
		exists  bool
		content string
	}

	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "true if self contained and contains alert manager config secret",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						AlertManagerConfigSecret: secretContent,
					},
				},
			},
			want: want{
				exists:  true,
				content: secretContent,
			},
		},
		{
			name: "false if spec is not self contained",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: nil,
				},
			},
			want: want{
				exists:  false,
				content: "",
			},
		},
		{
			name: "false if self contained and Blackbox Exporter enabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						AlertManagerConfigSecret: "",
					},
				},
			},
			want: want{
				exists:  false,
				content: "",
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &Observability{
				tt.fields.TypeMeta,
				tt.fields.ObjectMeta,
				tt.fields.Spec,
				tt.fields.Status,
			}
			exists, content := obs.HasAlertmanagerConfigSecret()
			Expect(exists).To(Equal(tt.want.exists))
			Expect(content).To(Equal(tt.want.content))
		})
	}
}

func TestObservabilityTypes_HasBlackboxBearerTokenSecret(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       ObservabilitySpec
		Status     ObservabilityStatus
	}

	type want struct {
		exists  bool
		content string
	}

	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "true if self contained and contains black box bearer token secret",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						BlackboxBearerTokenSecret: secretContent,
					},
				},
			},
			want: want{
				exists:  true,
				content: secretContent,
			},
		},
		{
			name: "false if not self contained",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: nil,
				},
			},
			want: want{
				exists:  false,
				content: "",
			},
		},
		{
			name: "false if self contained and Blackbox Exporter enabled",
			fields: fields{
				Spec: ObservabilitySpec{
					SelfContained: &SelfContained{
						BlackboxBearerTokenSecret: "",
					},
				},
			},
			want: want{
				exists:  false,
				content: "",
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &Observability{
				tt.fields.TypeMeta,
				tt.fields.ObjectMeta,
				tt.fields.Spec,
				tt.fields.Status,
			}
			exists, content := obs.HasBlackboxBearerTokenSecret()
			Expect(exists).To(Equal(tt.want.exists))
			Expect(content).To(Equal(tt.want.content))
		})
	}
}
