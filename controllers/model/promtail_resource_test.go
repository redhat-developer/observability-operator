package model

import (
	"testing"

	. "github.com/onsi/gomega"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testPromtailConfig = `
server:
  http_listen_port: 9080
  http_listen_address: 0.0.0.0
clients:
  - url: 
    external_labels:
      cluster_id: "test-cluster-id"
      observability_id: "test-observability"
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
          names: [test1,test2]
`
	testPromtailConfigDex = `
server:
  http_listen_port: 9080
  http_listen_address: 0.0.0.0
clients:
  - url: test-gateway/api/logs/v1/test-tenant/loki/api/v1/push
    bearer_token_file: /opt/secrets/token
    external_labels:
      cluster_id: "test-cluster-id"
      observability_id: "test-observability"
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
          names: [test1,test2]
`
	testPromtailConfigRedHat = `
server:
  http_listen_port: 9080
  http_listen_address: 0.0.0.0
clients:
  - url: http://token-refresher-logs-test-id.testNamespace.svc.cluster.local
    external_labels:
      cluster_id: "test-cluster-id"
      observability_id: "test-observability"
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
          names: [test1,test2]
`
)

func TestPromtailResources_GetPromtailConfigmap(t *testing.T) {
	type args struct {
		cr   *v1.Observability
		name string
	}

	tests := []struct {
		name string
		args args
		want *corev1.ConfigMap
	}{
		{
			name: "return Promtail ConfigMap",
			args: args{
				cr:   buildObservabilityCR(nil),
				name: "test",
			},
			want: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "promtail-config-test",
					Namespace: testNamespace,
					Labels: map[string]string{
						"managed-by": "observability-operator",
					},
				},
			},
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPromtailConfigmap(tt.args.cr, tt.args.name)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPromtailResources_GetPromtailDaemonSet(t *testing.T) {
	type args struct {
		cr   *v1.Observability
		name string
	}

	tests := []struct {
		name string
		args args
		want *appsv1.DaemonSet
	}{
		{
			name: "return Promtail daemon set",
			args: args{
				cr:   buildObservabilityCR(nil),
				name: "test",
			},
			want: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "promtail-test",
					Namespace: testNamespace,
					Labels: map[string]string{
						"managed-by": "observability-operator",
					},
				},
			},
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPromtailDaemonSet(tt.args.cr, tt.args.name)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPromtailResources_GetPromtailServiceAccount(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *corev1.ServiceAccount
	}{
		{
			name: "return Promtail service account",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obs-promtail",
					Namespace: testNamespace,
				},
			},
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPromtailServiceAccount(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPromtailResources_GetPromtailClusterRole(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *rbacv1.ClusterRole
	}{
		{
			name: "return Promtail cluster role",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "obs-promtail",
				},
			},
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPromtailClusterRole(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPromtailResources_GetPromtailClusterRoleBinding(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want *rbacv1.ClusterRoleBinding
	}{
		{
			name: "return Promtail cluster role binding",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "obs-promtail",
				},
			},
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPromtailClusterRoleBinding(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPromtailResources_GetPromtailConfig(t *testing.T) {
	type args struct {
		cr         *v1.Observability
		c          *v1.ObservatoriumIndex
		indexId    string
		namespaces []string
	}

	type wantErr struct {
		exists bool
		msg    string
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr wantErr
	}{
		{
			name: "return Promtail config when auth type is red hat",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Status.ClusterID = "test-cluster-id"
				}),
				c: &v1.ObservatoriumIndex{
					Id:       "test-id",
					Gateway:  "test-gateway",
					Tenant:   "test-tenant",
					AuthType: v1.AuthTypeRedhat,
					RedhatSsoConfig: &v1.RedhatSsoConfig{
						Url:        "test-url",
						Realm:      "test-realm",
						LogsSecret: "test-logs-secret",
						LogsClient: "test-logs-client",
					},
				},
				indexId:    "test-observability",
				namespaces: testPattern,
			},
			want: testPromtailConfigRedHat,
		},
		{
			name: "return Promtail config when no Observatorium index",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Status.ClusterID = "test-cluster-id"
				}),
				c:          nil,
				indexId:    "test-observability",
				namespaces: testPattern,
			},
			want: testPromtailConfig,
		},
		{
			name: "return Promtail config when auth type is dex",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Status.ClusterID = "test-cluster-id"
				}),
				c: &v1.ObservatoriumIndex{
					Id:       "test-id",
					Gateway:  "test-gateway",
					Tenant:   "test-tenant",
					AuthType: v1.AuthTypeDex,
				},
				indexId:    "test-observability",
				namespaces: testPattern,
			},
			want: testPromtailConfigDex,
		},

		{
			name: "error when Observatorium index is not valid",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Status.ClusterID = "test-cluster-id"
				}),
				c: &v1.ObservatoriumIndex{
					Id:      "test-id",
					Gateway: "",
					Tenant:  "",
				},
				indexId:    "test-observability",
				namespaces: testPattern,
			},
			want: "",
			wantErr: wantErr{
				exists: true,
				msg:    "invalid observatorium config for test-id",
			},
		},
		{
			name: "error when authtype is Red Hat SSO but config is nil",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Status.ClusterID = "test-cluster-id"
				}),
				c: &v1.ObservatoriumIndex{
					Id:              "test-id",
					Gateway:         "test-gateway",
					Tenant:          "test-tenant",
					AuthType:        v1.AuthTypeRedhat,
					RedhatSsoConfig: nil,
				},
				indexId:    "test-observability",
				namespaces: testPattern,
			},
			want: "",
			wantErr: wantErr{
				exists: true,
				msg:    "invalid sso config for test-id",
			},
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetPromtailConfig(tt.args.cr, tt.args.c, tt.args.indexId, tt.args.namespaces)
			Expect(err != nil).To(Equal(tt.wantErr.exists))
			if err != nil {
				Expect(err.Error()).To(Equal(tt.wantErr.msg))
			}
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPromtailResources_GetPromtailDaemonSetLabels(t *testing.T) {
	type args struct {
		index *v1.RepositoryIndex
	}

	tests := []struct {
		name string
		args args
		want *metav1.LabelSelector
	}{
		{
			name: "return DaemonSetLabelSelector from index",
			args: args{
				index: &testRepoIndexes[1],
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "promtail-test",
				},
			},
		},
		{
			name: "return DaemonSetLabelSelector from index",
			args: args{
				index: &v1.RepositoryIndex{
					Config: nil,
				},
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "promtail",
				},
			},
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPromtailDaemonSetLabels(tt.args.index)
			Expect(result).To(Equal(tt.want))
		})
	}
}
