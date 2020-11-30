package model

import (
	"bytes"
	"fmt"
	v12 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	routev1 "github.com/openshift/api/route/v1"
	v13 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	t "text/template"
)

func GetAlertmanagerProxySecret(cr *v1.Observability) *v13.Secret {
	return &v13.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "alertmanager-proxy",
			Namespace: cr.Name,
		},
	}
}

func GetAlertmanagerRoute(cr *v1.Observability) *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-alertmanager",
			Namespace: cr.Namespace,
		},
	}
}

func GetAlertmanagerServiceAccount(cr *v1.Observability) *v13.ServiceAccount {
	route := GetAlertmanagerRoute(cr)
	redirect := fmt.Sprintf("{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"%s\"}}", route.Name)

	return &v13.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-alertmanager",
			Namespace: cr.Namespace,
			Annotations: map[string]string{
				"serviceaccounts.openshift.io/oauth-redirectreference.primary": redirect,
			},
		},
	}
}

func GetAlertmanagerCr(cr *v1.Observability) *v12.Alertmanager {
	return &v12.Alertmanager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-alertmanager",
			Namespace: cr.Namespace,
		},
	}
}

func GetAlertmanagerSecret(cr *v1.Observability) *v13.Secret {
	alertmanager := GetAlertmanagerCr(cr)

	return &v13.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("alertmanager-%s", alertmanager.Name),
			Namespace: cr.Namespace,
		},
	}
}

func GetAlertmanagerService(cr *v1.Observability) *v13.Service {
	return &v13.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-alertmanager",
			Namespace: cr.Namespace,
		},
	}
}

func GetAlertmanagerConfig(cr *v1.Observability, secret string, url string) (string, error) {
	const config = `
global:
  resolve_timeout: 5m
route:
  receiver: default
  routes:
    - match:
        alertname: DeadMansSwitch
      repeat_interval: 5m
      receiver: deadmansswitch
receivers:
  - name: default
    pagerduty_configs:
      - service_key: {{ .PagerDutyServiceKey }}
  - name: deadmansswitch
    webhook_configs:
      - url: {{ .DeadMansSnitchURL }}
`
	template := t.Must(t.New("template").Parse(config))
	var buffer bytes.Buffer

	err := template.Execute(&buffer, struct {
		PagerDutyServiceKey string
		DeadMansSnitchURL   string
	}{
		PagerDutyServiceKey: secret,
		DeadMansSnitchURL:   url,
	})

	return string(buffer.Bytes()), err
}
