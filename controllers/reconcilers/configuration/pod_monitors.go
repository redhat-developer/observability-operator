package configuration

import (
	"context"
	"fmt"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	"github.com/ghodss/yaml"
	v12 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func MergeLabels(requested map[string]string, existing map[string]string) map[string]string {
	if existing == nil {
		return requested
	}

	for k, v := range requested {
		existing[k] = v
	}
	return existing
}

func getUniquePodMonitors(indexes []v1.RepositoryIndex) []ResourceInfo {
	var result []ResourceInfo
	for _, index := range indexes {
		if index.Config == nil || index.Config.Prometheus == nil {
			continue
		}
	seek:
		for _, monitor := range index.Config.Prometheus.PodMonitors {
			name := getNameFromUrl(monitor)
			for _, existing := range result {
				if existing.Name == name {
					continue seek
				}
			}
			result = append(result, ResourceInfo{
				Id:          index.Id,
				Name:        name,
				Url:         fmt.Sprintf("%s/%s", index.BaseUrl, monitor),
				AccessToken: index.AccessToken,
				Tag:         index.Tag,
			})
		}
	}
	return result
}

func (r *Reconciler) deleteUnrequestedPodMonitors(cr *v1.Observability, ctx context.Context, monitors []ResourceInfo) error {
	// List existing pod monitors
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

	// Check which pod monitors are no longer requested and delete them
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

func (r *Reconciler) createRequestedPodMonitors(cr *v1.Observability, ctx context.Context, monitors []ResourceInfo) error {
	// Sync requested pod monitors
	for _, resource := range monitors {
		bytes, err := r.fetchResource(resource.Url, resource.Tag, resource.AccessToken)
		if err != nil {
			return err
		}

		monitor, err := parsePodMonitorFromYaml(cr, resource.Name, bytes)
		if err != nil {
			return err
		}

		requestedLabels := monitor.Labels
		requestedSpec := monitor.Spec

		_, err = controllerutil.CreateOrUpdate(ctx, r.client, monitor, func() error {
			monitor.Spec = requestedSpec
			monitor.Labels = MergeLabels(map[string]string{
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

func parsePodMonitorFromYaml(cr *v1.Observability, name string, source []byte) (*v12.PodMonitor, error) {
	monitor := &v12.PodMonitor{}
	err := yaml.Unmarshal(source, monitor)
	if err != nil {
		return nil, err
	}
	monitor.Namespace = cr.Namespace
	monitor.Name = name
	return monitor, nil
}
