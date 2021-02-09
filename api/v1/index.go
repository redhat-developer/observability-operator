package v1

type RepositoryInfo struct {
	Repository  string
	Channel     string
	AccessToken string
}

type GrafanaIndex struct {
	Dashboards []string `json:"dashboards"`
}

type DexIndex struct {
	Url                       string `json:"url"`
	CredentialSecretName      string `json:"credentialSecretName"`
	CredentialSecretNamespace string `json:"credentialSecretNamespace"`
}

type ObservatoriumIndex struct {
	Gateway   string                `json:"gateway"`
	Tenant    string                `json:"tenant"`
	AuthType  ObservabilityAuthType `json:"authType"`
	DexConfig *DexConfig            `json:"dexConfig,omitempty"`
}

type RemoteWriteIndex struct {
	Patterns []string `json:"patterns"`
}

type AlertmanagerIndex struct {
	PagerDutySecretName           string `json:"pagerDutySecretName"`
	PagerDutySecretNamespace      string `json:"pagerDutySecretNamespace"`
	DeadmansSnitchSecretName      string `json:"deadmansSnitchSecretName"`
	DeadmansSnitchSecretNamespace string `json:"deadmansSnitchSecretNamespace"`
}

type PrometheusIndex struct {
	Rules         []string            `json:"rules"`
	PodMonitors   []string            `json:"pod_monitors"`
	Federation    string              `json:"federation,omitempty"`
	Observatorium *ObservatoriumIndex `json:"observatorium,omitempty"`
	RemoteWrite   string              `json:"remoteWrite,omitempty"`
}

type PromtailIndex struct {
	Enabled                bool              `json:"enabled,omitempty"`
	NamespaceLabelSelector map[string]string `json:"namespaceLabelSelector,omitempty"`
}

type RepositoryConfig struct {
	Grafana      *GrafanaIndex      `json:"grafana,omitempty"`
	Prometheus   *PrometheusIndex   `json:"prometheus,omitempty"`
	Alertmanager *AlertmanagerIndex `json:"alertmanager,omitempty"`
	Promtail     *PromtailIndex     `json:"promtail,omitempty"`
}

type RepositoryIndex struct {
	BaseUrl     string            `json:"-"`
	AccessToken string            `json:"-"`
	Id          string            `json:"id"`
	Config      *RepositoryConfig `json:"config"`
}
