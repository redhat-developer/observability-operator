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

type ResourceInfo struct {
	Id          string
	Name        string
	Url         string
	AccessToken string
	Tag         string
}

func getUniqueRules(indexes []v1.RepositoryIndex) []ResourceInfo {
	var result []ResourceInfo
seek:
	for _, index := range indexes {
		if index.Config == nil || index.Config.Prometheus == nil {
			continue
		}
		for _, rule := range index.Config.Prometheus.Rules {
			name := getNameFromUrl(rule)
			for _, existing := range result {
				if existing.Name == name {
					continue seek
				}
			}
			result = append(result, ResourceInfo{
				Id:          index.Id,
				Name:        name,
				Url:         fmt.Sprintf("%s/%s", index.BaseUrl, rule),
				AccessToken: index.AccessToken,
				Tag:         index.Tag,
			})
		}
	}
	return result
}

func (r *Reconciler) deleteUnrequestedRules(cr *v1.Observability, ctx context.Context, rules []ResourceInfo) error {
	// List existing dashboards
	existingRules := &v12.PrometheusRuleList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := r.client.List(ctx, existingRules, opts)
	if err != nil {
		return err
	}

	isRequested := func(name string) bool {
		for _, rule := range rules {
			if name == rule.Name {
				return true
			}
		}
		return false
	}

	// Check which rules are no longer requested and
	// delete them
	for _, rule := range existingRules.Items {
		if isRequested(rule.Name) == false {
			err = r.client.Delete(ctx, rule)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Reconciler) createRequestedRules(cr *v1.Observability, ctx context.Context, rules []ResourceInfo) error {
	// Sync requested prometheus rules
	for _, rule := range rules {
		bytes, err := r.fetchResource(rule.Url, rule.Tag, rule.AccessToken)
		if err != nil {
			return err
		}

		parsedRule, err := parseRuleFromYaml(cr, rule.Name, bytes)
		if err != nil {
			return err
		}

		requestedSpec := parsedRule.Spec
		requestedLabels := parsedRule.Labels

		_, err = controllerutil.CreateOrUpdate(ctx, r.client, parsedRule, func() error {
			// Add managed label to Rule CR
			parsedRule.Spec = requestedSpec
			parsedRule.Labels = MergeLabels(map[string]string{
				"managed-by": "observability-operator",
			}, requestedLabels)
			// Inject managed labels for each rule
			injectIdLabel(parsedRule, rule.Id)
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func injectIdLabel(rule *v12.PrometheusRule, id string) {
	for i := 0; i < len(rule.Spec.Groups); i++ {
		for j := 0; j < len(rule.Spec.Groups[i].Rules); j++ {
			if rule.Spec.Groups[i].Rules[j].Labels == nil {
				rule.Spec.Groups[i].Rules[j].Labels = make(map[string]string)
			}
			rule.Spec.Groups[i].Rules[j].Labels[PrometheusRuleIdentifierKey] = id
		}
	}
}

func parseRuleFromYaml(cr *v1.Observability, name string, source []byte) (*v12.PrometheusRule, error) {
	rule := &v12.PrometheusRule{}
	err := yaml.Unmarshal(source, rule)
	if err != nil {
		return nil, err
	}
	rule.Namespace = cr.Namespace
	rule.Name = name
	return rule, nil
}
