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
	DexConfig *DexConfig            `json:dexConfig,omitempty`
}

type PrometheusIndex struct {
	Rules         []string            `json:"rules"`
	Observatorium *ObservatoriumIndex `json:"observatorium,omitempty"`
}

type RepositoryConfig struct {
	Grafana    *GrafanaIndex    `json:"grafana,omitempty"`
	Prometheus *PrometheusIndex `json:"prometheus,omitempty"`
}

type RepositoryIndex struct {
	BaseUrl     string            `json:"-"`
	AccessToken string            `json:"-"`
	Id          string            `json:"id"`
	Config      *RepositoryConfig `json:"config"`
}
