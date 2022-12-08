package configuration

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	"github.com/redhat-developer/observability-operator/v4/controllers/model"
)

var (
	testGateway                   = "test-gateway"
	testTenant                    = "test-tenant"
	testSecret                    = "test-secret"
	testClient                    = "test-client"
	testRealm                     = "test-realm"
	testUrl                       = "test-url"
	testObservatoriumSsoConfigNil = &v1.ObservatoriumIndex{
		RedhatSsoConfig: nil,
	}
	testObservatoriumSsoConfigInvalidUrl = &v1.ObservatoriumIndex{
		RedhatSsoConfig: &v1.RedhatSsoConfig{
			Url: "invalid\nurl",
		},
	}
	testObservatoriumSsoConfigValidUrl = &v1.ObservatoriumIndex{
		RedhatSsoConfig: &v1.RedhatSsoConfig{
			Url: testUrl,
		},
	}
	testObservatoriumSsoConfigHasMetrics = &v1.ObservatoriumIndex{
		Id:      "test-id",
		Gateway: testGateway,
		Tenant:  testTenant,
		RedhatSsoConfig: &v1.RedhatSsoConfig{
			Url:           testUrl,
			Realm:         testRealm,
			MetricsClient: testClient,
			MetricsSecret: testSecret,
		},
	}
	testObservatoriumSsoConfigHasLogs = &v1.ObservatoriumIndex{
		Id:      "test-id",
		Gateway: testGateway,
		Tenant:  testTenant,
		RedhatSsoConfig: &v1.RedhatSsoConfig{
			Url:        testUrl,
			Realm:      testRealm,
			LogsClient: testClient,
			LogsSecret: testSecret,
		},
	}
)

func TestTokenRefresher_GetTokenRefresherConfigSetFor(t *testing.T) {
	type args struct {
		tokenRefresherType model.TokenRefresherType
		observatorium      *v1.ObservatoriumIndex
	}
	tests := []struct {
		name    string
		args    args
		want    *model.TokenRefresherConfigSet
		wantErr bool
	}{
		{
			name: "return nil if sso conifg is nil",
			args: args{
				observatorium: testObservatoriumSsoConfigNil,
			},
		},
		{
			name: "error if sso config url is invalid",
			args: args{
				observatorium: testObservatoriumSsoConfigInvalidUrl,
			},
			wantErr: true,
		},
		{
			name: "return nil if type is MetricsTokenRefresher and sso config has NO metrics",
			args: args{
				tokenRefresherType: model.MetricsTokenRefresher,
				observatorium:      testObservatoriumSsoConfigValidUrl,
			},
		},
		{
			name: "returns TokenRefresherConfigSet with metrics client and secret",
			args: args{
				tokenRefresherType: model.MetricsTokenRefresher,
				observatorium:      testObservatoriumSsoConfigHasMetrics,
			},
			want: &model.TokenRefresherConfigSet{
				ObservatoriumUrl: fmt.Sprintf("%v/api/metrics/v1/%v/api/v1/receive", testGateway, testTenant),
				AuthUrl:          "test-url/realms/test-realm",
				Name:             "token-refresher-metrics-test-id",
				Realm:            testRealm,
				Tenant:           testTenant,
				Secret:           testSecret,
				Client:           testClient,
				Type:             model.MetricsTokenRefresher,
			},
		},
		{
			name: "return nil if type is LogsTokenRefresher and sso config has NO logs",
			args: args{
				tokenRefresherType: model.LogsTokenRefresher,
				observatorium:      testObservatoriumSsoConfigValidUrl,
			},
		},
		{
			name: "returns TokenRefresherConfigSet with logs client and secret",
			args: args{
				tokenRefresherType: model.LogsTokenRefresher,
				observatorium:      testObservatoriumSsoConfigHasLogs,
			},
			want: &model.TokenRefresherConfigSet{
				ObservatoriumUrl: fmt.Sprintf("%v/api/logs/v1/%v/loki/api/v1/push", testGateway, testTenant),
				AuthUrl:          "test-url/realms/test-realm",
				Name:             "token-refresher-logs-test-id",
				Realm:            testRealm,
				Tenant:           testTenant,
				Secret:           testSecret,
				Client:           testClient,
				Type:             model.LogsTokenRefresher,
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getTokenRefresherConfigSetFor(tt.args.tokenRefresherType, tt.args.observatorium)
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(result).To(Equal(tt.want))
		})
	}
}
