package configuration

import (
	"context"
	"fmt"
	"github.com/ghodss/yaml"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/model"
	v12 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) reconcileAlertmanagerSecret(ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex) error {
	root := &v1.AlertmanagerConfigRoute{
		Receiver: "default-receiver",
		Routes:   []v1.AlertmanagerConfigRoute{},
	}

	config := v1.AlertmanagerConfigRoot{
		Global: &v1.AlertmanagerConfigGlobal{
			ResolveTimeout: "5m",
		},
		Route: root,
		Receivers: []v1.AlertmanagerConfigReceiver{
			{
				Name: "default-receiver",
			},
		},
	}

	for _, index := range indexes {
		if index.Config == nil || index.Config.Alertmanager == nil {
			continue
		}

		pagerDutySecret, err := r.getPagerDutySecret(ctx, cr, index.Config.Alertmanager)
		if err != nil {
			continue
		}

		deadmansSnitchUrl, err := r.getDeadMansSnitchUrl(ctx, cr, index.Config.Alertmanager)
		if err != nil {
			continue
		}

		pagerDutyReceiver := fmt.Sprintf("%s-%s", index.Id, "pagerduty")
		deadMansSnitchReceiver := fmt.Sprintf("%s-%s", index.Id, "deadmanssnitch")

		config.Receivers = append(config.Receivers, v1.AlertmanagerConfigReceiver{
			Name: pagerDutyReceiver,
			PagerDutyConfigs: []v1.PagerDutyConfig{
				{
					ServiceKey: string(pagerDutySecret),
				},
			},
		})

		config.Receivers = append(config.Receivers, v1.AlertmanagerConfigReceiver{
			Name: deadMansSnitchReceiver,
			WebhookConfigs: []v1.WebhookConfig{
				{
					Url: string(deadmansSnitchUrl),
				},
			},
		})

		root.Routes = append(root.Routes, v1.AlertmanagerConfigRoute{
			Receiver: pagerDutyReceiver,
			Match: map[string]string{
				"severity":                  "critical",
				PrometheusRuleIdentifierKey: index.Id,
			},
		})

		root.Routes = append(root.Routes, v1.AlertmanagerConfigRoute{
			Receiver: deadMansSnitchReceiver,
			Match: map[string]string{
				"alertname":                 "DeadMansSwitch",
				PrometheusRuleIdentifierKey: index.Id,
			},
		})
	}

	configBytes, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}

	secret := model.GetAlertmanagerSecret(cr)

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, secret, func() error {
		secret.Type = v12.SecretTypeOpaque
		secret.StringData = map[string]string{
			"alertmanager.yaml": string(configBytes),
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) getPagerDutySecret(ctx context.Context, cr *v1.Observability, config *v1.AlertmanagerIndex) ([]byte, error) {
	if config == nil {
		return []byte("dummy"), nil
	}

	if config.PagerDutySecretName == "" {
		return []byte("dummy"), nil
	}

	ns := cr.Namespace
	if config.PagerDutySecretNamespace != "" {
		ns = config.PagerDutySecretNamespace
	}

	pagerdutySecret := &v12.Secret{}
	selector := client.ObjectKey{
		Namespace: ns,
		Name:      config.PagerDutySecretName,
	}
	err := r.client.Get(ctx, selector, pagerdutySecret)
	if err != nil {
		return nil, err
	}

	var secret []byte
	if len(pagerdutySecret.Data["PAGERDUTY_KEY"]) != 0 {
		secret = pagerdutySecret.Data["PAGERDUTY_KEY"]
	} else if len(pagerdutySecret.Data["serviceKey"]) != 0 {
		secret = pagerdutySecret.Data["serviceKey"]
	}

	return secret, nil
}

func (r *Reconciler) getDeadMansSnitchUrl(ctx context.Context, cr *v1.Observability, config *v1.AlertmanagerIndex) ([]byte, error) {
	if config == nil {
		return []byte("http://dummy"), nil
	}

	if config.DeadmansSnitchSecretName == "" {
		return []byte("http://dummy"), nil
	}

	ns := cr.Namespace
	if config.DeadmansSnitchSecretNamespace != "" {
		ns = config.DeadmansSnitchSecretNamespace
	}

	dmsSecret := &v12.Secret{}
	selector := client.ObjectKey{
		Namespace: ns,
		Name:      config.DeadmansSnitchSecretName,
	}
	err := r.client.Get(ctx, selector, dmsSecret)
	if err != nil {
		return nil, err
	}

	var url []byte
	if len(dmsSecret.Data["SNITCH_URL"]) != 0 {
		url = dmsSecret.Data["SNITCH_URL"]
	} else if len(dmsSecret.Data["url"]) != 0 {
		url = dmsSecret.Data["url"]
	}

	return url, nil
}
