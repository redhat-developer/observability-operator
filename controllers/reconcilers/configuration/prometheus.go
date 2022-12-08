package configuration

import (
	"context"
	"fmt"
	"regexp"

	"github.com/ghodss/yaml"
	errors2 "github.com/pkg/errors"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	"github.com/redhat-developer/observability-operator/v4/controllers/model"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers/token"
	"github.com/redhat-developer/observability-operator/v4/controllers/utils"
	"github.com/sirupsen/logrus"
	kv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	PrometheusBaseImage = "quay.io/prometheus/prometheus"
	PrometheusRetention = "45d"
)

func (r *Reconciler) fetchFederationConfigs(cr *v1.Observability, indexes []v1.RepositoryIndex) ([]string, error) {
	var result []string

	type federationPatterns struct {
		Match []string `json:"match[]"`
	}

	hasPattern := func(newPattern string) bool {
		for _, pattern := range result {
			if newPattern == pattern {
				return true
			}
		}
		return false
	}

	// Allow to specify federated metrics in CR when external repo sync is disabled
	if cr.ExternalSyncDisabled() {
		return cr.Spec.SelfContained.FederatedMetrics, nil
	}

	for _, index := range indexes {
		if index.Config == nil || index.Config.Prometheus == nil || index.Config.Prometheus.Federation == "" {
			continue
		}

		federationConfigUrl := fmt.Sprintf("%s/%s", index.BaseUrl, index.Config.Prometheus.Federation)
		bytes, err := r.fetchResource(federationConfigUrl, index.Tag, index.AccessToken)
		if err != nil {
			return nil, err
		}

		var indexConfig federationPatterns
		err = yaml.Unmarshal(bytes, &indexConfig)
		if err != nil {
			return nil, err
		}

		for _, pattern := range indexConfig.Match {
			if !hasPattern(pattern) {
				result = append(result, fmt.Sprintf("'%s'", pattern))
			}
		}
	}

	return result, nil
}

func (r *Reconciler) createBlackBoxConfig(cr *v1.Observability, ctx context.Context) (string, error) {
	configMap := model.GetPrometheusBlackBoxConfig(cr)
	cfg, hash, err := model.GetDefaultBlackBoxConfig(cr, ctx, r.client)
	if err != nil {
		return hash, err
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, configMap, func() error {
		configMap.Data = map[string]string{
			"black-box-config.yaml": string(cfg),
		}
		return nil
	})
	return hash, err
}

// Write the additional scrape config secret, used to federate from openshift-monitoring
// This expects the aggregation of all federation configs across all indexes
func (r *Reconciler) createAdditionalScrapeConfigSecret(cr *v1.Observability, ctx context.Context, patterns []string) error {
	secret := model.GetPrometheusAdditionalScrapeConfig(cr)
	federationConfig, err := model.GetFederationConfigBearerToken(patterns)
	if err != nil {
		return err
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, secret, func() error {
		secret.Type = kv1.SecretTypeOpaque
		secret.StringData = map[string]string{
			"additional-scrape-config.yaml": string(federationConfig),
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) getRemoteWriteIndex(index v1.RepositoryIndex) (*v1.RemoteWriteIndex, error) {
	patternUrl := fmt.Sprintf("%s/%s", index.BaseUrl, index.Config.Prometheus.RemoteWrite)
	bytes, err := r.fetchResource(patternUrl, index.Tag, index.AccessToken)
	if err != nil {
		return nil, err
	}

	remoteWrite := v1.RemoteWriteIndex{}
	err = yaml.Unmarshal(bytes, &remoteWrite)
	if err != nil {
		return nil, errors2.Wrap(err, "error parsing remote write index")
	}
	return &remoteWrite, nil
}

// Send requests directly to observatorium
func (r *Reconciler) getRemoteWriteSpecForDex(index v1.RepositoryIndex, observatoriumConfig *v1.ObservatoriumIndex, remoteWrite *v1.RemoteWriteIndex) (*prometheusv1.RemoteWriteSpec, string, error) {
	tokenSecret := token.GetObservatoriumPrometheusSecretName(&index)
	return &prometheusv1.RemoteWriteSpec{
		URL:                 fmt.Sprintf("%s/api/metrics/v1/%s/api/v1/receive", observatoriumConfig.Gateway, observatoriumConfig.Tenant),
		Name:                index.Id,
		RemoteTimeout:       prometheusv1.Duration(remoteWrite.RemoteTimeout),
		WriteRelabelConfigs: remoteWrite.WriteRelabelConfigs,
		BearerTokenFile:     fmt.Sprintf("/etc/prometheus/secrets/%s/token", tokenSecret),
		TLSConfig: &prometheusv1.TLSConfig{
			SafeTLSConfig: prometheusv1.SafeTLSConfig{
				InsecureSkipVerify: true,
			},
		},
		ProxyURL:    remoteWrite.ProxyUrl,
		QueueConfig: remoteWrite.QueueConfig,
	}, tokenSecret, nil
}

// Proxy requests through the token refresher
func (r *Reconciler) getRemoteWriteSpecForRedHat(cr *v1.Observability, index v1.RepositoryIndex, observatoriumConfig *v1.ObservatoriumIndex, remoteWrite *v1.RemoteWriteIndex) (*prometheusv1.RemoteWriteSpec, string, error) {
	tokenRefresherName := model.GetTokenRefresherName(observatoriumConfig.Id, model.MetricsTokenRefresher)
	tokenRefresherUrl := fmt.Sprintf("http://%v.%v.svc.cluster.local", tokenRefresherName, cr.GetPrometheusOperatorNamespace())

	return &prometheusv1.RemoteWriteSpec{
		URL:                 tokenRefresherUrl,
		Name:                index.Id,
		RemoteTimeout:       prometheusv1.Duration(remoteWrite.RemoteTimeout),
		WriteRelabelConfigs: remoteWrite.WriteRelabelConfigs,
		TLSConfig: &prometheusv1.TLSConfig{
			SafeTLSConfig: prometheusv1.SafeTLSConfig{
				InsecureSkipVerify: true,
			},
		},
		ProxyURL:    remoteWrite.ProxyUrl,
		QueueConfig: remoteWrite.QueueConfig,
	}, "", nil
}

func (r *Reconciler) getRemoteWriteSpec(cr *v1.Observability, index v1.RepositoryIndex, remoteWrite *v1.RemoteWriteIndex) (*prometheusv1.RemoteWriteSpec, string, error) {
	if index.Config == nil || index.Config.Prometheus == nil || index.Config.Prometheus.Observatorium == "" {
		return nil, "", fmt.Errorf("no observatorium config found for %v / prometheus", index.Id)
	}

	observatoriumConfig := token.GetObservatoriumConfig(&index, index.Config.Prometheus.Observatorium)
	if observatoriumConfig == nil {
		return nil, "", fmt.Errorf("no observatorium config found for %v", index.Config.Prometheus.Observatorium)
	}

	switch observatoriumConfig.AuthType {
	case v1.AuthTypeDex:
		return r.getRemoteWriteSpecForDex(index, observatoriumConfig, remoteWrite)
	case v1.AuthTypeRedhat:
		return r.getRemoteWriteSpecForRedHat(cr, index, observatoriumConfig, remoteWrite)
	default:
		return nil, "", errors2.New(fmt.Sprintf("unknown auth type %v", observatoriumConfig.AuthType))
	}
}

func (r *Reconciler) getAlerting(cr *v1.Observability) *prometheusv1.AlertingSpec {
	alertmanager := model.GetAlertmanagerCr(cr)
	alertmanagerService := model.GetAlertmanagerService(cr)

	return &prometheusv1.AlertingSpec{
		Alertmanagers: []prometheusv1.AlertmanagerEndpoints{
			{
				Namespace: cr.GetPrometheusOperatorNamespace(),
				Name:      alertmanager.Name,
				Port:      intstr.FromString("web"),
				Scheme:    "https",
				TLSConfig: &prometheusv1.TLSConfig{
					CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt",
					SafeTLSConfig: prometheusv1.SafeTLSConfig{
						ServerName: fmt.Sprintf("%v.%v.svc", alertmanagerService.Name, cr.GetPrometheusOperatorNamespace()),
					},
				},
				BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
			},
		},
	}
}

func (r *Reconciler) reconcilePrometheus(ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex, configHash string) error {
	proxySecret := model.GetPrometheusProxySecret(cr)
	sa := model.GetPrometheusServiceAccount(cr)

	route := model.GetPrometheusRoute(cr)
	selector := client.ObjectKey{
		Namespace: route.Namespace,
		Name:      route.Name,
	}

	err := r.client.Get(ctx, selector, route)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	host := ""
	if utils.IsRouteReady(route) {
		host = route.Spec.Host
	}

	var secrets []string
	secrets = append(secrets, proxySecret.Name)
	secrets = append(secrets, "prometheus-k8s-tls")

	var remoteWrites []prometheusv1.RemoteWriteSpec
	var sidecars []kv1.Container

	// If Observatorium is disabled, we won't create any remote write targets
	if !cr.ObservatoriumDisabled() {
		for _, index := range indexes {
			rw, err := r.getRemoteWriteIndex(index)
			if err != nil {
				return err
			}

			remoteWrite, tokenSecret, err := r.getRemoteWriteSpec(cr, index, rw)
			if err != nil {
				logrus.Error(err)
				continue
			}

			remoteWrites = append(remoteWrites, *remoteWrite)
			if tokenSecret != "" {
				secrets = append(secrets, tokenSecret)
			}
		}
	}

	var image = fmt.Sprintf("%s:%s", PrometheusBaseImage, model.GetPrometheusVersion(cr))

	sidecars = append(sidecars, kv1.Container{
		Name:  "oauth-proxy",
		Image: "quay.io/openshift/origin-oauth-proxy:4.8",
		Args: []string{
			"-provider=openshift",
			"-https-address=:9091",
			"-http-address=",
			"-email-domain=*",
			"-upstream=http://localhost:9090",
			fmt.Sprintf("-openshift-service-account=%v", sa.Name),
			"-openshift-sar={\"resource\": \"namespaces\", \"verb\": \"get\"}",
			"-openshift-delegate-urls={\"/\": {\"resource\": \"namespaces\", \"verb\": \"get\"}}",
			"-tls-cert=/etc/tls/private/tls.crt",
			"-tls-key=/etc/tls/private/tls.key",
			"-client-secret-file=/var/run/secrets/kubernetes.io/serviceaccount/token",
			"-cookie-secret-file=/etc/proxy/secrets/session_secret",
			"-openshift-ca=/etc/pki/tls/cert.pem",
			"-openshift-ca=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
			"-skip-auth-regex=^/metrics",
		},
		Env: []kv1.EnvVar{
			{
				Name: "HTTP_PROXY",
			},
			{
				Name: "HTTPS_PROXY",
			},
			{
				Name: "NO_PROXY",
			},
		},
		Ports: []kv1.ContainerPort{
			{
				Name:          "proxy",
				ContainerPort: 9091,
			},
		},
		VolumeMounts: []kv1.VolumeMount{
			{
				Name:      "secret-prometheus-k8s-tls",
				MountPath: "/etc/tls/private",
			},
			{
				Name:      fmt.Sprintf("secret-%v", proxySecret.Name),
				MountPath: "/etc/proxy/secrets",
			},
		},
	})

	if !cr.BlackboxExporterDisabled() {
		sidecars = append(sidecars, kv1.Container{
			Name:  "blackbox-exporter",
			Image: "quay.io/prometheus/blackbox-exporter:v0.19.0",
			Args: []string{
				"--config.file=/opt/config/black-box-config.yaml",
			},
			Env: []kv1.EnvVar{
				{
					Name:  "CONFIG_HASH",
					Value: configHash,
				},
			},
			Ports: []kv1.ContainerPort{
				{
					Name:          "http",
					ContainerPort: 9115,
				},
			},
			VolumeMounts: []kv1.VolumeMount{
				{
					Name:      "black-box-config",
					MountPath: "/opt/config/",
				},
				{
					Name:      "secret-prometheus-k8s-tls",
					MountPath: "/etc/tls/private",
				},
			},
		})
	}
	prometheus := model.GetPrometheus(cr)
	_, err = controllerutil.CreateOrUpdate(ctx, r.client, prometheus, func() error {
		cr.Labels = map[string]string{
			"app": "prometheus",
		}

		prometheus.Spec = prometheusv1.PrometheusSpec{
			CommonPrometheusFields: prometheusv1.CommonPrometheusFields{
				PodMetadata: &prometheusv1.EmbeddedObjectMetadata{
					Annotations: map[string]string{
						"cluster-autoscaler.kubernetes.io/safe-to-evict": "true",
					},
				},
				// Custom Prometheus version
				Image:   &image,
				Version: model.GetPrometheusVersion(cr),

				PriorityClassName: model.ObservabilityPriorityClassName,

				// Spec
				ServiceAccountName: sa.Name,
				ExternalURL:        fmt.Sprintf("https://%v", host),
				AdditionalScrapeConfigs: &kv1.SecretKeySelector{
					LocalObjectReference: kv1.LocalObjectReference{
						Name: "additional-scrape-configs",
					},
					Key: "additional-scrape-config.yaml",
				},
				ExternalLabels: map[string]string{
					"cluster_id": cr.Status.ClusterID,
				},
				Volumes: []kv1.Volume{
					{
						Name: "black-box-config",
						VolumeSource: kv1.VolumeSource{
							ConfigMap: &kv1.ConfigMapVolumeSource{
								LocalObjectReference: kv1.LocalObjectReference{
									Name: "black-box-config",
								},
							},
						},
					},
				},
				PodMonitorSelector:              model.GetPrometheusPodMonitorLabelSelectors(cr, indexes),
				PodMonitorNamespaceSelector:     model.GetPrometheusPodMonitorNamespaceSelectors(cr, indexes),
				ServiceMonitorSelector:          model.GetPrometheusServiceMonitorLabelSelectors(cr, indexes),
				ServiceMonitorNamespaceSelector: model.GetPrometheusServiceMonitorNamespaceSelectors(cr, indexes),

				ProbeSelector:          model.GetProbeLabelSelectors(cr, indexes),
				ProbeNamespaceSelector: model.GetProbeNamespaceSelectors(cr, indexes),
				RemoteWrite:            remoteWrites,

				Secrets:    secrets,
				Containers: sidecars,
				Resources:  *model.GetPrometheusResourceRequirement(cr),
			},
			Retention:             getRetentionHelper(cr),
			RuleSelector:          model.GetPrometheusRuleLabelSelectors(cr, indexes),
			RuleNamespaceSelector: model.GetPrometheusRuleNamespaceSelectors(cr, indexes),
			Alerting:              r.getAlerting(cr),
		}
		if cr.Spec.Storage != nil && cr.Spec.Storage.PrometheusStorageSpec != nil {
			var prometheusStorageSpec *prometheusv1.StorageSpec
			existingPV, pvName, err := r.existingPVC(cr, ctx)
			if err != nil {
				return err
			}
			if existingPV {
				prometheusStorageSpec, err = r.useExistingPVForVolumeClaim(pvName, ctx, cr, indexes)
				if err != nil {
					return err
				}
			} else {
				prometheusStorageSpec, err = getPrometheusStorageSpecHelper(cr, indexes)
				if err != nil {
					return err
				}
			}
			prometheus.Spec.Storage = prometheusStorageSpec
		}
		if cr.Spec.Tolerations != nil {
			prometheus.Spec.Tolerations = cr.Spec.Tolerations
		}
		if cr.Spec.Affinity != nil {
			prometheus.Spec.Affinity = cr.Spec.Affinity
		}
		return nil
	})

	if err != nil {
		return err
	}

	// need to remove the unbound PVC once new PVC is bound to existing PV
	err = r.removePVCPostMigration(ctx, cr)
	if err != nil {
		return err
	}
	return nil
}

// check for existing PVC
func (r *Reconciler) existingPVC(cr *v1.Observability, ctx context.Context) (bool, string, error) {
	var exists bool
	pvc := kv1.PersistentVolumeClaim{}
	pvcList := &kv1.PersistentVolumeClaimList{}
	opts := &client.ListOptions{
		Namespace: cr.GetPrometheusOperatorNamespace(),
	}
	err := r.client.List(ctx, pvcList, opts)
	if err != nil {
		return exists, "", errors2.Wrap(err, "error listing existing volume claims")
	}
	for _, pvc = range pvcList.Items {
		if pvc.Name == "managed-services-prometheus-kafka-prometheus-0" {
			exists = true
		}
	}
	return exists, pvc.Spec.VolumeName, nil
}

// use existing PV for PVC
func (r *Reconciler) useExistingPVForVolumeClaim(volumeName string, ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex) (*prometheusv1.StorageSpec, error) {
	prometheusStorageSpec := cr.Spec.Storage.PrometheusStorageSpec
	pv := &kv1.PersistentVolume{}
	selector := client.ObjectKey{
		Name: volumeName,
	}
	err := r.client.Get(ctx, selector, pv)
	if err != nil {
		return prometheusStorageSpec, err
	}

	pv.Spec.ClaimRef = &kv1.ObjectReference{
		Name:      "managed-services-prometheus-obs-prometheus-0",
		Namespace: cr.GetPrometheusOperatorNamespace(),
	}

	err = r.client.Update(ctx, pv)
	prometheusStorageSpec = &prometheusv1.StorageSpec{
		VolumeClaimTemplate: prometheusv1.EmbeddedPersistentVolumeClaim{
			EmbeddedObjectMetadata: prometheusv1.EmbeddedObjectMetadata{
				Name: "managed-services",
			},
			Spec: kv1.PersistentVolumeClaimSpec{
				VolumeName: volumeName,
				Resources: kv1.ResourceRequirements{
					Requests: pv.Spec.Capacity,
				},
			},
		},
	}

	return prometheusStorageSpec, err
}

// remove redundant PVC once new PVC is bound to existing volume
func (r *Reconciler) removePVCPostMigration(ctx context.Context, cr *v1.Observability) error {
	pvc := &kv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "managed-services-prometheus-kafka-prometheus-0",
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
	}

	err := r.client.Delete(ctx, pvc)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}

// construct Prometheus storage spec with either default or override value from resources
func getPrometheusStorageSpecHelper(cr *v1.Observability, indexes []v1.RepositoryIndex) (*prometheusv1.StorageSpec, error) {
	prometheusStorageSpec := cr.Spec.Storage.PrometheusStorageSpec
	if cr.ExternalSyncDisabled() {
		return prometheusStorageSpec, nil
	}

	customStorageSize := model.GetPrometheusStorageSize(cr, indexes)
	if customStorageSize == "" {
		return prometheusStorageSpec, nil
	}
	parsedQuantity, err := resource.ParseQuantity(customStorageSize) //check if resources value is valid
	if err == nil {
		prometheusStorageSpec = &prometheusv1.StorageSpec{
			VolumeClaimTemplate: prometheusv1.EmbeddedPersistentVolumeClaim{
				EmbeddedObjectMetadata: prometheusv1.EmbeddedObjectMetadata{
					Name: "managed-services",
				},
				Spec: kv1.PersistentVolumeClaimSpec{
					Resources: kv1.ResourceRequirements{
						Requests: map[kv1.ResourceName]resource.Quantity{kv1.ResourceStorage: parsedQuantity},
					},
				},
			},
		}
	}
	return prometheusStorageSpec, err
}

func getRetentionHelper(cr *v1.Observability) prometheusv1.Duration {
	match, err := regexp.MatchString("^[0-9]+(((ms)|y|w|d|h|m|s)){1}$", cr.Spec.Retention)
	if err != nil || !match {
		return prometheusv1.Duration(PrometheusRetention)
	}

	return prometheusv1.Duration(cr.Spec.Retention)
}
