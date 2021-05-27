package configuration

import (
	"context"
	"fmt"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	"github.com/ghodss/yaml"
	"github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	url2 "net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

type SourceType int

const (
	SourceTypeJson    SourceType = 1
	SourceTypeJsonnet SourceType = 2
	SourceTypeYaml    SourceType = 3
	SourceTypeUnknown SourceType = 4
)

type DashboardInfo struct {
	Name        string
	Url         string
	AccessToken string
	Tag         string
}

func getNameFromUrl(path string) string {
	parts := strings.Split(path, string(types.Separator))
	part := parts[len(parts)-1]
	parts = strings.Split(part, ".")
	return parts[0]
}

func getUniqueDashboards(indexes []v1.RepositoryIndex) []DashboardInfo {
	var result []DashboardInfo
seek:
	for _, index := range indexes {
		if index.Config == nil || index.Config.Grafana == nil {
			continue
		}
		for _, dashboard := range index.Config.Grafana.Dashboards {
			name := getNameFromUrl(dashboard)
			for _, existing := range result {
				if existing.Name == name {
					continue seek
				}
			}
			result = append(result, DashboardInfo{
				Name:        name,
				Url:         fmt.Sprintf("%s/%s", index.BaseUrl, dashboard),
				AccessToken: index.AccessToken,
				Tag:         index.Tag,
			})
		}
	}
	return result
}

func (r *Reconciler) deleteUnrequestedDashboards(cr *v1.Observability, ctx context.Context, dashboards []DashboardInfo) error {
	// List existing dashboards
	existingDashboards := &v1alpha1.GrafanaDashboardList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := r.client.List(ctx, existingDashboards, opts)
	if err != nil {
		return err
	}

	isRequested := func(name string) bool {
		for _, dashboard := range dashboards {
			if name == dashboard.Name {
				return true
			}
		}
		return false
	}

	// Check which dashboards are no longer requested and
	// delete them
	for _, dashboard := range existingDashboards.Items {
		if isRequested(dashboard.Name) == false {
			err = r.client.Delete(ctx, &dashboard)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Reconciler) createRequestedDashboards(cr *v1.Observability, ctx context.Context, dashboards []DashboardInfo) error {
	// Create a list of requested dashboards from the external sources provided
	// in the CR
	var requestedDashboards []*v1alpha1.GrafanaDashboard
	for _, d := range dashboards {
		sourceType, source, err := r.fetchDashboard(d.Url, d.Tag, d.AccessToken)
		if err != nil {
			return err
		}

		switch sourceType {
		case SourceTypeUnknown:
			break
		case SourceTypeYaml:
			dashboard, err := parseDashboardFromYaml(cr, d.Name, source)
			if err != nil {
				return err
			}
			requestedDashboards = append(requestedDashboards, dashboard)
		case SourceTypeJsonnet:
		case SourceTypeJson:
			dashboard, err := createDashboardFromSource(cr, d.Name, sourceType, source)
			if err != nil {
				return err
			}
			requestedDashboards = append(requestedDashboards, dashboard)
		default:
		}
	}

	// Sync requested dashboards
	for _, dashboard := range requestedDashboards {

		requestedSpec := dashboard.Spec
		requestedLabels := dashboard.Labels

		_, err := controllerutil.CreateOrUpdate(ctx, r.client, dashboard, func() error {
			dashboard.Spec = requestedSpec
			dashboard.Labels = MergeLabels(map[string]string{
				"managed-by": "observability-operator",
			}, requestedLabels)
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func parseDashboardFromYaml(cr *v1.Observability, name string, source []byte) (*v1alpha1.GrafanaDashboard, error) {
	dashboard := &v1alpha1.GrafanaDashboard{}
	err := yaml.Unmarshal(source, dashboard)
	if err != nil {
		return nil, err
	}
	dashboard.Namespace = cr.Namespace
	dashboard.Name = name
	return dashboard, nil
}

func createDashboardFromSource(cr *v1.Observability, name string, t SourceType, source []byte) (*v1alpha1.GrafanaDashboard, error) {
	dashboard := &v1alpha1.GrafanaDashboard{}
	dashboard.Name = name
	dashboard.Namespace = cr.Namespace
	dashboard.Spec.Name = fmt.Sprintf("%s.json", name)

	switch t {
	case SourceTypeJson:
		dashboard.Spec.Json = string(source)
	case SourceTypeJsonnet:
		dashboard.Spec.Jsonnet = string(source)
	default:
		return nil, fmt.Errorf("unknown dashboard type: %v", name)
	}

	return dashboard, nil
}

// Try to determine the type (json or grafonnet) or a remote file by looking
// at the filename extension
func getFileType(path string) SourceType {
	fragments := strings.Split(path, ".")
	if len(fragments) == 0 {
		return SourceTypeUnknown
	}

	extension := strings.TrimSpace(fragments[len(fragments)-1])
	switch strings.ToLower(extension) {
	case "json":
		return SourceTypeJson
	case "grafonnet":
		return SourceTypeJsonnet
	case "jsonnet":
		return SourceTypeJsonnet
	case "yaml":
		return SourceTypeYaml
	default:
		return SourceTypeUnknown
	}
}

func (r *Reconciler) fetchDashboard(path string, tag string, token string) (SourceType, []byte, error) {
	url, err := url2.ParseRequestURI(path)
	if err != nil {
		return SourceTypeUnknown, nil, err
	}

	if token == "" {
		return SourceTypeUnknown, nil, fmt.Errorf("repository ConfigMap missing required AccessToken")
	}

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return SourceTypeUnknown, nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	if tag != "" {
		q := req.URL.Query()
		q.Add("ref", tag)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return SourceTypeUnknown, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return SourceTypeUnknown, nil, fmt.Errorf("unexpected status code: %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return SourceTypeUnknown, nil, err
	}

	sourceType := getFileType(url.Path)
	return sourceType, body, nil
}
