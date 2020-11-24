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

const (
	GrafanaInstallation     ObservabilityStageName = "Grafana"
	GrafanaConfiguration    ObservabilityStageName = "GrafanaConfiguration"
	PrometheusInstallation  ObservabilityStageName = "Prometheus"
	PrometheusConfiguration ObservabilityStageName = "PrometheusConfiguration"
	PrometheusRules         ObservabilityStageName = "PrometheusRules"
	CsvRemoval              ObservabilityStageName = "CsvRemoval"
)

const (
	ResultSuccess    ObservabilityStageStatus = "success"
	ResultFailed     ObservabilityStageStatus = "failed"
	ResultInProgress ObservabilityStageStatus = "in progress"
)

type ObservatoriumConfig struct {
	Gateway string `json:"gateway"`
	Token   string `json:"token"`
	Tenant  string `json:"tenant"`
}

// ObservabilitySpec defines the desired state of Observability
type ObservabilitySpec struct {
	Observatorium *ObservatoriumConfig `json:"observatorium,omitempty"`
}

// ObservabilityStatus defines the observed state of Observability
type ObservabilityStatus struct {
	Stage       ObservabilityStageName   `json:"stage"`
	StageStatus ObservabilityStageStatus `json:"stageStatus"`
	LastMessage string                   `json:"lastMessage,omitempty"`
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
