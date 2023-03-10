package v1

import (
	"testing"

	. "github.com/onsi/gomega"
)

var (
	configUrl   = "red-hat-sso-config-url"
	configRealm = "red-hat-sso-config-realm"
)

func TestIndex_HasAuthServer(t *testing.T) {
	type fields struct {
		Url           string
		Realm         string
		MetricsClient string
		MetricsSecret string
		LogsClient    string
		LogsSecret    string
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "true if config contains url and realm",
			fields: fields{
				Url:   configUrl,
				Realm: configRealm,
			},
			want: true,
		},
		{
			name: "false if config contains no url",
			fields: fields{
				Url:   "",
				Realm: configRealm,
			},
			want: false,
		},
		{
			name: "false if config contains no realm",
			fields: fields{
				Url:   configUrl,
				Realm: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)

			config := &RedhatSsoConfig{
				tt.fields.Url,
				tt.fields.Realm,
				tt.fields.MetricsClient,
				tt.fields.MetricsSecret,
				tt.fields.LogsClient,
				tt.fields.LogsSecret,
			}

			result := config.HasAuthServer()
			g.Expect(result).To(Equal(tt.want))
		})
	}
}

func TestIndex_HasMetrics(t *testing.T) {
	type fields struct {
		Url           string
		Realm         string
		MetricsClient string
		MetricsSecret string
		LogsClient    string
		LogsSecret    string
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "true if config contains metrics client and secret",
			fields: fields{
				Url:           configUrl,
				Realm:         configRealm,
				MetricsClient: "metrics-client",
				MetricsSecret: "metrics-secret",
			},
			want: true,
		},
		{
			name: "false if config no metrics client",
			fields: fields{
				Url:           configUrl,
				Realm:         configRealm,
				MetricsClient: "",
				MetricsSecret: "metrics-secret",
			},
			want: false,
		},
		{
			name: "false if config no metrics secret",
			fields: fields{
				Url:           configUrl,
				Realm:         configRealm,
				MetricsClient: "metrics-client",
				MetricsSecret: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)

			config := &RedhatSsoConfig{
				tt.fields.Url,
				tt.fields.Realm,
				tt.fields.MetricsClient,
				tt.fields.MetricsSecret,
				tt.fields.LogsClient,
				tt.fields.LogsSecret,
			}

			result := config.HasMetrics()
			g.Expect(result).To(Equal(tt.want))
		})
	}
}

func TestIndex_HasLogs(t *testing.T) {
	type fields struct {
		Url           string
		Realm         string
		MetricsClient string
		MetricsSecret string
		LogsClient    string
		LogsSecret    string
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "true if config contains logs client and secret",
			fields: fields{
				Url:        configUrl,
				Realm:      configRealm,
				LogsClient: "logs-client",
				LogsSecret: "logs-secret",
			},
			want: true,
		},
		{
			name: "false if config no metrics client",
			fields: fields{
				Url:        configUrl,
				Realm:      configRealm,
				LogsClient: "",
				LogsSecret: "logs-secret",
			},
			want: false,
		},
		{
			name: "false if config no metrics client",
			fields: fields{
				Url:        configUrl,
				Realm:      configRealm,
				LogsClient: "logs-client",
				LogsSecret: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)

			config := &RedhatSsoConfig{
				tt.fields.Url,
				tt.fields.Realm,
				tt.fields.MetricsClient,
				tt.fields.MetricsSecret,
				tt.fields.LogsClient,
				tt.fields.LogsSecret,
			}

			result := config.HasLogs()
			g.Expect(result).To(Equal(tt.want))
		})
	}
}

func TestIndex_IsValid(t *testing.T) {
	type fields struct {
		Id              string
		SecretName      string
		Gateway         string
		Tenant          string
		AuthType        ObservabilityAuthType
		DexConfig       *DexConfig
		RedhatSsoConfig *RedhatSsoConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "true if contains gateway and tenant",
			fields: fields{
				Gateway: "gateway",
				Tenant:  "tenant",
			},
			want: true,
		},
		{
			name: "false if contains no gateway",
			fields: fields{
				Gateway: "",
				Tenant:  "tenant",
			},
			want: false,
		},
		{
			name: "true if contains no tenant",
			fields: fields{
				Gateway: "gateway",
				Tenant:  "",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)

			obsIndex := &ObservatoriumIndex{
				tt.fields.Id,
				tt.fields.SecretName,
				tt.fields.Gateway,
				tt.fields.Tenant,
				tt.fields.AuthType,
				tt.fields.DexConfig,
				tt.fields.RedhatSsoConfig,
			}

			result := obsIndex.IsValid()
			g.Expect(result).To(Equal(tt.want))
		})
	}
}
