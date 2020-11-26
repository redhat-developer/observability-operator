package model

import (
	"bytes"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	t "text/template"
)

func GetPromtailConfigmap(cr *v1.Observability) *v12.ConfigMap {
	return &v12.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "promtaila-config",
			Namespace: cr.Namespace,
		},
	}
}

func GetPromtailConfig(cr *v1.Observability, id string, namespaces []string) (string, error) {
	const config = `
    server:
      http_listen_port: 9080
      http_listen_address: 0.0.0.0
	external_labels:
		cluster_id: "{{ .ClusterID }}"
    clients:
      - url: {{ .Url }}
        bearer_token: {{ .Token }}
        tls_config:
          insecure_skip_verify: true
    scrape_configs:
      - job_name: "strimzi"
        relabel_configs:
        - source_labels:
          - __meta_kubernetes_pod_node_name
          target_label: __host__
        - action: replace
          source_labels:
          - __meta_kubernetes_pod_node_name
          target_label: nodename
        - action: replace
          source_labels:
          - __meta_kubernetes_namespace
          target_label: namespace
        - action: replace
          source_labels:
          - __meta_kubernetes_pod_name
          target_label: instance
        - action: replace
          source_labels:
          - __meta_kubernetes_pod_container_name
          target_label: container_name
        - action: labelmap
          regex: __meta_kubernetes_pod_label_strimzi_io_(.+)
          replacement: strimzi_io_$1
        - replacement: /var/log/pods/*$1/*.log
          separator: /
          source_labels:
          - __meta_kubernetes_pod_uid
          - __meta_kubernetes_pod_container_name
          target_label: __path__
        kubernetes_sd_configs:
          - role: "pod"
            namespaces:
              names: [{{ .Namespaces }}]
`
	template := t.Must(t.New("template").Parse(config))
	var buffer bytes.Buffer

	url := ""
	if cr.Spec.Observatorium != nil {

	}

	err := template.Execute(&buffer, struct {
		ClusterID  string
		Namespaces string
		Url        string
		Token      string
	}{
		ClusterID:  id,
		Namespaces: strings.Join(namespaces, ","),
		Url:        url,
		Token:      "",
	})

	return string(buffer.Bytes()), err
}
