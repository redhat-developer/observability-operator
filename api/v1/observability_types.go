/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ObservabilityStageName string

type ObservabilityStageStatus string

type ObservabilityAuthType string

const (
	GrafanaInstallation      ObservabilityStageName = "Grafana"
	GrafanaConfiguration     ObservabilityStageName = "GrafanaConfiguration"
	PrometheusInstallation   ObservabilityStageName = "Prometheus"
	PrometheusConfiguration  ObservabilityStageName = "PrometheusConfiguration"
	PrometheusRules          ObservabilityStageName = "PrometheusRules"
	CsvRemoval               ObservabilityStageName = "CsvRemoval"
	TokenRequest             ObservabilityStageName = "TokenRequest"
	PromtailInstallation     ObservabilityStageName = "PromtailInstallation"
	AlertmanagerInstallation ObservabilityStageName = "AlertmanagerInstallation"
)

const (
	ResultSuccess    ObservabilityStageStatus = "success"
	ResultFailed     ObservabilityStageStatus = "failed"
	ResultInProgress ObservabilityStageStatus = "in progress"
)

const (
	AuthTypeDex ObservabilityAuthType = "dex"
)

type DexConfig struct {
	Url                       string `json:"url"`
	CredentialSecretNamespace string `json:"credentialSecretNamespace"`
	CredentialSecretName      string `json:"credentialSecretName"`
}

type DashboardSource struct {
	Url  string `json:"url"`
	Name string `json:"name"`
}

type GrafanaConfig struct {
	// Dashboards to create from external sources
	Dashboards []*DashboardSource `json:"dashboards,omitempty"`

	// How often to refetch the dashboards?
	ResyncPeriod string `json:"resyncPeriod,omitempty"`

	// If false, the operator will install default dashboards and ignore list
	Managed bool `json:"managed"`
}

type ObservatoriumConfig struct {
	// Observatorium Gateway API URL
	Gateway string `json:"gateway"`
	// Observatorium tenant name
	Tenant string `json:"tenant"`

	// Auth type. Currently only dex is supported
	AuthType ObservabilityAuthType `json:"authType,omitempty"`

	// Dex configuration
	AuthDex *DexConfig `json:"dexConfig,omitempty"`
}

type AlertmanagerConfig struct {
	PagerDutySecretName           string `json:"pagerDutySecretName"`
	PagerDutySecretNamespace      string `json:"pagerDutySecretNamespace,omitempty"`
	DeadMansSnitchSecretName      string `json:"deadMansSnitchSecretName"`
	DeadMansSnitchSecretNamespace string `json:"deadMansSnitchSecretNamespace,omitempty"`
}

// ObservabilitySpec defines the desired state of Observability
type ObservabilitySpec struct {
	// Observatorium config
	Observatorium *ObservatoriumConfig `json:"observatorium,omitempty"`

	// Grafana config
	Grafana *GrafanaConfig `json:"grafana,omitempty"`

	// Alertmanager config
	Alertmanager *AlertmanagerConfig `json:"alertmanager,omitempty"`

	// Selector for all namespaces that should be scraped
	KafkaNamespaceSelector *metav1.LabelSelector `json:"kafkaNamespaceSelector,omitempty"`

	// Selector for all canary pods that should be scraped
	CanaryPodSelector *metav1.LabelSelector `json:"canaryPodSelector,omitempty"`

	// Cluster ID. If not provided, the operator tries to obtain it.
	ClusterID string `json:"clusterId,omitempty"`
}

// ObservabilityStatus defines the observed state of Observability
type ObservabilityStatus struct {
	Stage                ObservabilityStageName   `json:"stage"`
	StageStatus          ObservabilityStageStatus `json:"stageStatus"`
	LastMessage          string                   `json:"lastMessage,omitempty"`
	TokenExpires         int64                    `json:"tokenExpires,omitempty"`
	ClusterID            string                   `json:"clusterId,omitempty"`
	DashboardsLastSynced int64                    `json:"dashboardsLastSynced,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Observability is the Schema for the observabilities API
type Observability struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ObservabilitySpec   `json:"spec,omitempty"`
	Status ObservabilityStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ObservabilityList contains a list of Observability
type ObservabilityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Observability `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Observability{}, &ObservabilityList{})
}
