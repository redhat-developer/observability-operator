package configuration

import (
	"context"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"io/ioutil"
	"net/http"
	url2 "net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type SourceType int

const (
	SourceTypeJson    SourceType = 1
	SourceTypeJsonnet SourceType = 2
	SourceTypeYaml    SourceType = 3
	SourceTypeUnknown SourceType = 4
)

func getUniqueDashboards(indexes []RepositoryIndex) []string {
	var result []string
	for _, index := range indexes {
		if index.Config == nil || index.Config.Grafana == nil {
			continue
		}
		for _, dashboard := range index.Config.Grafana.Dashboards {
			for _, existing := range result {
				if existing == dashboard {
					continue
				}
			}
			result = append(result, dashboard)
		}
	}
	return result
}

func deleteUnrequestedDashboards(cr *v1.Observability, ctx context.Context, c client.Client, dashboards []string) error {
	// List existing dashboards
	existingDashboards := &v1alpha1.GrafanaDashboardList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := c.List(ctx, existingDashboards, opts)
	if err != nil {
		return err
	}

	isRequested := func(name string) bool {
		for _, dashboard := range dashboards {
			if name == dashboard {
				return true
			}
		}
		return false
	}

	// Check which dashboards are no longer requested and
	// delete them
	for _, dashboard := range existingDashboards.Items {
		if isRequested(dashboard.Name) == false {
			err = c.Delete(ctx, &dashboard)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func createRequestedDashboards(cr *v1.Observability, ctx context.Context, c client.Client, baseUrl string, dashboards []string) error {
	// Create a list of requested dashboards from the external sources provided
	// in the CR
	var requestedDashboards []*v1alpha1.GrafanaDashboard
	for _, d := range dashboards {
		dashboardUrl := fmt.Sprintf("")
		sourceType, source, err := r.fetchDashboard(d.Url)
		if err != nil {
			return v1.ResultFailed, err
		}

		switch sourceType {
		case SourceTypeUnknown:
			break
		case SourceTypeYaml:
			dashboard, err := r.parseDashboardFromYaml(cr, d.Name, source)
			if err != nil {
				return v1.ResultFailed, err
			}
			requestedDashboards = append(requestedDashboards, dashboard)
		case SourceTypeJsonnet:
		case SourceTypeJson:
			dashboard, err := r.createDashbaordFromSource(cr, d.Name, sourceType, source)
			if err != nil {
				return v1.ResultFailed, err
			}
			requestedDashboards = append(requestedDashboards, dashboard)
		default:
		}
	}
}


func (r *Reconciler) parseDashboardFromYaml(cr *v1.Observability, name string, source []byte) (*v1alpha1.GrafanaDashboard, error) {
	dashboard := &v1alpha1.GrafanaDashboard{}
	err := yaml.Unmarshal(source, dashboard)
	if err != nil {
		return nil, err
	}
	dashboard.Namespace = cr.Namespace
	dashboard.Name = name
	return dashboard, nil
}

func (r *Reconciler) createDashbaordFromSource(cr *v1.Observability, name string, t SourceType, source []byte) (*v1alpha1.GrafanaDashboard, error) {
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
func (r *Reconciler) getFileType(path string) SourceType {
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

func (r *Reconciler) fetchDashboard(path string) (SourceType, []byte, error) {
	url, err := url2.ParseRequestURI(path)
	if err != nil {
		return SourceTypeUnknown, nil, err
	}

	resp, err := http.Get(url.String())
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

	sourceType := r.getFileType(url.Path)
	return sourceType, body, nil
}
