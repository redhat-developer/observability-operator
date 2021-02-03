package configuration

import (
	"context"
	"fmt"
	v12 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/ghodss/yaml"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func getUniquePodMonitors(indexes []v1.RepositoryIndex) []ResourceInfo {
	var result []ResourceInfo
	for _, index := range indexes {
		if index.Config == nil || index.Config.Prometheus == nil {
			continue
		}
		for _, rule := range index.Config.Prometheus.PodMonitors {
			name := getNameFromUrl(rule)
			for _, existing := range result {
				if existing.Name == name {
					continue
				}
			}
			result = append(result, ResourceInfo{
				Id:          index.Id,
				Name:        name,
				Url:         fmt.Sprintf("%s/%s", index.BaseUrl, rule),
				AccessToken: index.AccessToken,
			})
		}
	}
	return result
}

func (r *Reconciler) deleteUnrequestedPodMonitors(cr *v1.Observability, ctx context.Context, monitors []ResourceInfo) error {
	// List existing dashboards
	existingMonitors := &v12.PodMonitorList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := r.client.List(ctx, existingMonitors, opts)
	if err != nil {
		return err
	}

	isRequested := func(name string) bool {
		for _, monitor := range monitors {
			if name == monitor.Name {
				return true
			}
		}
		return false
	}

	// Check which rules are no longer requested and
	// delete them
	for _, monitor := range existingMonitors.Items {
		if isRequested(monitor.Name) == false {
			err = r.client.Delete(ctx, monitor)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Reconciler) createRequestedPodMonitors(cr *v1.Observability, ctx context.Context, rules []ResourceInfo) error {
	// Sync requested prometheus rules
	for _, rule := range rules {
		bytes, err := r.fetchResource(rule.Url, rule.AccessToken)
		if err != nil {
			return err
		}

		parsedRule, err := parseRuleFromYaml(cr, rule.Name, bytes)
		if err != nil {
			return err
		}

		_, err = controllerutil.CreateOrUpdate(ctx, r.client, parsedRule, func() error {
			// Inject managed labels
			injectIdLabel(parsedRule, rule.Id)
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func parsePodMonitorFromYaml(cr *v1.Observability, name string, source []byte) (*v12.PrometheusRule, error) {
	rule := &v12.PrometheusRule{}
	err := yaml.Unmarshal(source, rule)
	if err != nil {
		return nil, err
	}
	rule.Namespace = cr.Namespace
	rule.Name = name
	return rule, nil
}
