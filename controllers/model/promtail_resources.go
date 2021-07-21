package model

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	t "text/template"

	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	errors2 "github.com/pkg/errors"
	v13 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	v14 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetPromtailConfigmap(cr *v1.Observability, name string) *v12.ConfigMap {
	return &v12.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("promtail-config-%s", name),
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"managed-by": "observability-operator",
			},
		},
	}
}

func GetPromtailDaemonSet(cr *v1.Observability, name string) *v13.DaemonSet {
	return &v13.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("promtail-%s", name),
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"managed-by": "observability-operator",
			},
		},
	}
}

func GetPromtailServiceAccount(cr *v1.Observability) *v12.ServiceAccount {
	return &v12.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-promtail",
			Namespace: cr.Namespace,
		},
	}
}

func GetPromtailClusterRole(cr *v1.Observability) *v14.ClusterRole {
	return &v14.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kafka-promtail",
		},
	}
}

func GetPromtailClusterRoleBinding(cr *v1.Observability) *v14.ClusterRoleBinding {
	return &v14.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kafka-promtail",
		},
	}
}

func GetPromtailConfig(cr *v1.Observability, c *v1.ObservatoriumIndex, indexId string, namespaces []string) (string, error) {
	const config = `
server:
  http_listen_port: 9080
  http_listen_address: 0.0.0.0
clients:
  - url: {{ .Url }}
	{{- if .RequireToken }}
    bearer_token_file: /opt/secrets/token
	{{- end }}
    external_labels:
      cluster_id: "{{ .ClusterID }}"
      observability_id: "{{ .ObservabililtyId }}"
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
	var requireToken = false
	var buffer bytes.Buffer
	var url string

	if c != nil {
		if !c.IsValid() {
			return "", errors2.New(fmt.Sprintf("invalid observatorium config for %v", c.Id))
		}
		switch c.AuthType {
		case v1.AuthTypeDex:
			url = fmt.Sprintf("%s/api/logs/v1/%s/loki/api/v1/push", c.Gateway, c.Tenant)
			requireToken = true
		case v1.AuthTypeRedhat:
			if c.RedhatSsoConfig == nil || !c.RedhatSsoConfig.HasLogs() {
				return "", errors2.New(fmt.Sprintf("invalid sso config for %v", c.Id))
			}
			tokenRefresherName := GetTokenRefresherName(c.Id, LogsTokenRefresher)
			url = fmt.Sprintf("http://%v.%v.svc.cluster.local", tokenRefresherName, cr.Namespace)
		}
	}

	// Namespaces must be ordered to avoid different config hashes
	sort.Strings(namespaces)

	err := template.Execute(&buffer, struct {
		ClusterID        string
		ObservabililtyId string
		Namespaces       string
		Url              string
		RequireToken     bool
	}{
		ClusterID:        cr.Status.ClusterID,
		ObservabililtyId: indexId,
		Namespaces:       strings.Join(namespaces, ","),
		Url:              url,
		RequireToken:     requireToken,
	})

	return string(buffer.Bytes()), err
}

func GetPromtailDaemonSetLabels(index *v1.RepositoryIndex) *metav1.LabelSelector {
	if index.Config != nil && index.Config.Promtail != nil && index.Config.Promtail.DaemonSetLabelSelector != nil {
		return index.Config.Promtail.DaemonSetLabelSelector
	}
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": "promtail",
		},
	}
}
