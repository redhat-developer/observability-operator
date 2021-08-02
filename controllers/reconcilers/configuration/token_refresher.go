package configuration

import (
	"context"
	"fmt"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/controllers/model"
	errors2 "github.com/pkg/errors"
	v13 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	v15 "k8s.io/api/networking/v1"
	v14 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"net/url"
	"path"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	TokenRefresherImageTag = "master-2021-06-03-a835a06"
)

// Return a set of credentials and configuration for either logs or metrics
func getTokenRefresherConfigSetFor(t model.TokenRefresherType, observatorium *v1.ObservatoriumIndex) (*model.TokenRefresherConfigSet, error) {
	if observatorium.RedhatSsoConfig == nil {
		return nil, nil
	}

	result := &model.TokenRefresherConfigSet{}
	result.Name = model.GetTokenRefresherName(observatorium.Id, t)

	authEndpoint, err := url.Parse(observatorium.RedhatSsoConfig.Url)
	if err != nil {
		return nil, err
	}

	authEndpoint.Path = path.Join(authEndpoint.Path, "realms", observatorium.RedhatSsoConfig.Realm)

	result.AuthUrl = authEndpoint.String()
	result.Realm = observatorium.RedhatSsoConfig.Realm
	result.Tenant = observatorium.Tenant
	result.Type = t
	switch t {
	case model.MetricsTokenRefresher:
		if !observatorium.RedhatSsoConfig.HasMetrics() {
			return nil, nil
		}

		result.ObservatoriumUrl = fmt.Sprintf("%v/api/metrics/v1/%v/api/v1/receive", observatorium.Gateway, observatorium.Tenant)
		result.Secret = observatorium.RedhatSsoConfig.MetricsSecret
		result.Client = observatorium.RedhatSsoConfig.MetricsClient
	case model.LogsTokenRefresher:
		if !observatorium.RedhatSsoConfig.HasLogs() {
			return nil, nil
		}

		result.ObservatoriumUrl = fmt.Sprintf("%v/api/logs/v1/%v/loki/api/v1/push", observatorium.Gateway, observatorium.Tenant)
		result.Secret = observatorium.RedhatSsoConfig.LogsSecret
		result.Client = observatorium.RedhatSsoConfig.LogsClient
	default:
		return nil, nil
	}
	return result, nil
}

func (r *Reconciler) createServiceFor(ctx context.Context, cr *v1.Observability, config *model.TokenRefresherConfigSet) error {
	service := model.GetTokenRefresherService(cr, config.Name)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, service, func() error {
		service.Spec.Ports = []v12.ServicePort{
			{
				Name:        "http",
				Protocol:    "",
				AppProtocol: nil,
				Port:        80,
				TargetPort: intstr.IntOrString{
					IntVal: 8080,
				},
				NodePort: 0,
			},
		}
		service.Spec.Selector = map[string]string{
			"app.kubernetes.io/component": "authentication-proxy",
			"app.kubernetes.io/name":      config.Name,
		}
		return nil
	})

	return err
}

func (r *Reconciler) createNetworkPolicyFor(ctx context.Context, cr *v1.Observability, config *model.TokenRefresherConfigSet) error {
	policy := model.GetTokenRefresherNetworkPolicy(cr, config.Name)

	selector := make(map[string]string)
	switch config.Type {
	case model.LogsTokenRefresher:
		selector["app"] = "promtail"
	case model.MetricsTokenRefresher:
		selector["app"] = "prometheus"
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, policy, func() error {
		policy.Labels = map[string]string{
			"app.kubernetes.io/component": "authentication-proxy",
		}
		policy.Spec = v15.NetworkPolicySpec{
			PodSelector: v14.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component": "authentication-proxy",
					"app.kubernetes.io/name":      config.Name,
				},
			},
			Ingress: []v15.NetworkPolicyIngressRule{
				{
					From: []v15.NetworkPolicyPeer{
						{
							PodSelector: &v14.LabelSelector{
								MatchLabels: selector,
							},
						},
					},
				},
			},
			PolicyTypes: []v15.PolicyType{v15.PolicyTypeIngress},
		}
		return nil
	})

	return err
}

func (r *Reconciler) createDeploymentFor(ctx context.Context, cr *v1.Observability, config *model.TokenRefresherConfigSet) error {
	deployment := model.GetTokenRefresherDeployment(cr, config.Name)

	err := r.createNetworkPolicyFor(ctx, cr, config)
	if err != nil {
		return err
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, deployment, func() error {
		deployment.Labels = map[string]string{
			"app.kubernetes.io/component": "authentication-proxy",
			"app.kubernetes.io/name":      config.Name,
		}
		deployment.Spec = v13.DeploymentSpec{
			Selector: &v14.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component": "authentication-proxy",
					"app.kubernetes.io/name":      config.Name,
				},
			},
			Template: v12.PodTemplateSpec{
				ObjectMeta: v14.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/component": "authentication-proxy",
						"app.kubernetes.io/name":      config.Name,
						"app.kubernetes.io/version":   TokenRefresherImageTag,
					},
				},
				Spec: v12.PodSpec{
					PriorityClassName: model.ObservabilityPriorityClassName,
					Containers: []v12.Container{
						{
							Name:  config.Name,
							Image: fmt.Sprintf("quay.io/observatorium/token-refresher:%v", TokenRefresherImageTag),
							Args: []string{
								"--oidc.audience=observatorium-telemeter",
								fmt.Sprintf("--oidc.client-id=%v", config.Client),
								fmt.Sprintf("--oidc.client-secret=%v", config.Secret),
								fmt.Sprintf("--oidc.issuer-url=%v", config.AuthUrl),
								fmt.Sprintf("--url=%v", config.ObservatoriumUrl),
							},
							Ports: []v12.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		}
		return nil
	})

	return err
}

func (r *Reconciler) reconcileTokenRefresherFor(ctx context.Context, cr *v1.Observability, observatorium *v1.ObservatoriumIndex, logsDisabled bool) error {
	if !observatorium.IsValid() {
		return errors2.New(fmt.Sprintf("incomplete observatorium config, tenant or gateway missing for %v", observatorium.Id))
	}

	for _, t := range []model.TokenRefresherType{model.MetricsTokenRefresher, model.LogsTokenRefresher} {
		// Don't deploy a token refresher for promtail when logs are disabled
		if t == model.LogsTokenRefresher && logsDisabled {
			continue
		}

		configSet, err := getTokenRefresherConfigSetFor(t, observatorium)
		if err != nil {
			return err
		}

		if configSet == nil {
			// Do not abort in case of error, setups that skip logs are expected
			r.logger.Info(fmt.Sprintf("skip creating %v token refresher for %v because of missing config", t, observatorium.Id))
			continue
		}

		err = r.createServiceFor(ctx, cr, configSet)
		if err != nil {
			return err
		}

		err = r.createDeploymentFor(ctx, cr, configSet)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) reconcileTokenRefresher(ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex) error {
	for _, index := range indexes {
		if index.Config == nil {
			continue
		}

		promtailDisabled := false
		if index.Config.Promtail == nil || index.Config.Promtail.Enabled == false {
			promtailDisabled = true
		}

		for _, observatorium := range index.Config.Observatoria {
			// token-refresher is only used for sso.redhat.com authentication
			if observatorium.AuthType == v1.AuthTypeRedhat {
				err := r.reconcileTokenRefresherFor(ctx, cr, &observatorium, promtailDisabled)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (r *Reconciler) deleteUnrequestedNetworkPolicies(ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex) error {
	list := &v15.NetworkPolicyList{}
	selector := labels.SelectorFromSet(map[string]string{
		"app.kubernetes.io/component": "authentication-proxy",
	})
	opts := &client.ListOptions{
		Namespace:     cr.Namespace,
		LabelSelector: selector,
	}

	err := r.client.List(ctx, list, opts)
	if err != nil {
		return err
	}

	shouldExist := func(name string) bool {
		if cr.ExternalSyncDisabled() || cr.ObservatoriumDisabled() {
			return false
		}

		for _, index := range indexes {
			if index.Config == nil {
				return false
			}

			for _, observatorium := range index.Config.Observatoria {
				if !observatorium.IsValid() || observatorium.RedhatSsoConfig == nil {
					return false
				}

				for _, t := range []model.TokenRefresherType{model.MetricsTokenRefresher, model.LogsTokenRefresher} {
					if t == model.LogsTokenRefresher && (index.Config.Promtail == nil || !index.Config.Promtail.Enabled) {
						return false
					}

					configSet, err := getTokenRefresherConfigSetFor(t, &observatorium)
					if err != nil {
						return false
					}

					if name == fmt.Sprintf("%v-network-policy", configSet.Name) {
						return true
					}
				}
			}
		}
		return false
	}

	for _, policy := range list.Items {
		if !shouldExist(policy.Name) {
			err = r.client.Delete(ctx, &policy)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Reconciler) deleteUnrequestedTokenRefreshers(ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex) error {
	list := &v13.DeploymentList{}
	selector := labels.SelectorFromSet(map[string]string{
		"app.kubernetes.io/component": "authentication-proxy",
	})
	opts := &client.ListOptions{
		Namespace:     cr.Namespace,
		LabelSelector: selector,
	}

	err := r.client.List(ctx, list, opts)
	if err != nil {
		return err
	}

	shouldExist := func(name string) bool {
		if cr.ExternalSyncDisabled() || cr.ObservatoriumDisabled() {
			return false
		}

		for _, index := range indexes {
			if index.Config == nil {
				return false
			}

			for _, observatorium := range index.Config.Observatoria {
				if !observatorium.IsValid() || observatorium.RedhatSsoConfig == nil {
					return false
				}

				for _, t := range []model.TokenRefresherType{model.MetricsTokenRefresher, model.LogsTokenRefresher} {
					if t == model.LogsTokenRefresher && (index.Config.Promtail == nil || !index.Config.Promtail.Enabled) {
						return false
					}

					configSet, err := getTokenRefresherConfigSetFor(t, &observatorium)
					if err != nil {
						return false
					}

					if name == configSet.Name {
						return true
					}
				}
			}
		}
		return false
	}

	for _, deployment := range list.Items {
		if !shouldExist(deployment.Name) {
			err = r.client.Delete(ctx, &deployment)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
