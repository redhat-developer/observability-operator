package configuration

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	v14 "k8s.io/api/networking/v1"

	"github.com/go-logr/logr"
	"github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	errors2 "github.com/pkg/errors"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	"github.com/redhat-developer/observability-operator/v4/controllers/metrics"
	"github.com/redhat-developer/observability-operator/v4/controllers/model"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers"
	token2 "github.com/redhat-developer/observability-operator/v4/controllers/reconcilers/token"
	"github.com/redhat-developer/observability-operator/v4/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RemoteRepository            = "repository"
	RemoteAccessToken           = "access_token"
	RemoteChannel               = "channel"
	RemoteTag                   = "tag"
	PrometheusRuleIdentifierKey = "observability"
	DefaultChannel              = "resources"
)

type Reconciler struct {
	client     client.Client
	logger     logr.Logger
	httpClient *http.Client
}

func NewReconciler(client client.Client, logger logr.Logger) reconcilers.ObservabilityReconciler {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	return &Reconciler{
		client:     client,
		logger:     logger,
		httpClient: httpClient,
	}
}

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"managed-by": "observability-operator",
		}),
		Namespace: cr.Namespace,
	}

	list := &corev1.SecretList{}
	err := r.client.List(ctx, list, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	// Delete all managed secrets
	for _, secret := range list.Items {
		err := r.client.Delete(ctx, &secret)
		if err != nil {
			return v1.ResultFailed, err
		}
	}

	configMapList := &corev1.ConfigMapList{}
	err = r.client.List(ctx, configMapList, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	// Delete all managed config maps
	for _, configmap := range configMapList.Items {
		err := r.client.Delete(ctx, &configmap)
		if err != nil {
			return v1.ResultFailed, err
		}
	}

	if !cr.ExternalSyncDisabled() {
		if !cr.DescopedModeEnabled() {
			dashboardList := &v1alpha1.GrafanaDashboardList{}
			err = r.client.List(ctx, dashboardList, opts)
			if err != nil {
				return v1.ResultFailed, err
			}

			// Delete all managed grafana dashboards
			for _, dashboard := range dashboardList.Items {
				err := r.client.Delete(ctx, &dashboard)
				if err != nil {
					return v1.ResultFailed, err
				}
			}
		}

		prometheusRuleList := &prometheusv1.PrometheusRuleList{}
		err = r.client.List(ctx, prometheusRuleList, opts)
		if err != nil {
			return v1.ResultFailed, err
		}

		// Delete all managed prometheus rules
		for _, rule := range prometheusRuleList.Items {
			err := r.client.Delete(ctx, rule)
			if err != nil {
				return v1.ResultFailed, err
			}
		}

		podMonitorList := &prometheusv1.PodMonitorList{}
		err = r.client.List(ctx, podMonitorList, opts)
		if err != nil {
			return v1.ResultFailed, err
		}

		// Delete all managed pod monitors
		for _, monitor := range podMonitorList.Items {
			err := r.client.Delete(ctx, monitor)
			if err != nil {
				return v1.ResultFailed, err
			}
		}
	}

	// Delete Promtail daemonsets
	daemonsetList := &appsv1.DaemonSetList{}
	err = r.client.List(ctx, daemonsetList, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	for _, daemonset := range daemonsetList.Items {
		err := r.client.Delete(ctx, &daemonset)
		if err != nil {
			return v1.ResultFailed, err
		}
	}

	// Delete token refresher deployments
	tokenRefreshers := &appsv1.DeploymentList{}
	opts = &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"app.kubernetes.io/component": "authentication-proxy",
		}),
		Namespace: cr.Namespace,
	}
	err = r.client.List(ctx, tokenRefreshers, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	for _, deployment := range tokenRefreshers.Items {
		err := r.client.Delete(ctx, &deployment)
		if err != nil {
			return v1.ResultFailed, err
		}
	}

	// Delete resources deployment
	resourceDeployment := model.GetResourcesDeployment(cr)
	err = r.client.Delete(ctx, resourceDeployment)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	// Delete resources service
	service := model.GetResourcesService(cr)
	err = r.client.Delete(ctx, service)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	resourcesRoute := model.GetResourcesRoute(cr)
	err = r.client.Delete(ctx, resourcesRoute)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	// Delete the network policies
	networkPolicies := &v14.NetworkPolicyList{}
	opts = &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"app.kubernetes.io/component": "authentication-proxy",
		}),
		Namespace: cr.Namespace,
	}
	err = r.client.List(ctx, networkPolicies, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	for _, policy := range networkPolicies.Items {
		err := r.client.Delete(ctx, &policy)
		if err != nil {
			return v1.ResultFailed, err
		}
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) stampConfigSource(ctx context.Context, index *v1.RepositoryIndex) error {
	if index.Source != nil {
		// Update source secret
		if index.Source.Annotations == nil {
			index.Source.Annotations = map[string]string{}
		}
		index.Source.Annotations["observability-operator/status"] = "accepted"
		err := r.client.Update(ctx, index.Source)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	var containerResources bool

	log := r.logger.WithValues("observability", cr.Name)
	if cr.Spec.ConfigurationSelector == nil && !cr.ExternalSyncDisabled() {
		log.Info("warning: configuration label selector not present, dynamic configuration will be skipped")
		return v1.ResultSuccess, nil
	}

	// Force a sync if one of the tokens has expired
	overrideLastSync := false
	overrideLastSync, err := token2.TokensExpired(ctx, r.client, cr)
	if err != nil {
		metrics.IncreaseFailedConfigurationSyncsMetric()
		return v1.ResultFailed, errors2.Wrap(err, "error checking observatorium token lifetimes")
	}

	// Always react to CR updates when external repo sync is disabled
	if cr.ExternalSyncDisabled() {
		overrideLastSync = true
	}

	// Force resync if any CRs missing post-migration
	if s.Migrated {
		var missingCR bool
		missingCR, err = r.checkForMissingCR(ctx, cr)
		if err != nil {
			return v1.ResultFailed, errors2.Wrap(err, "error checking for migrated CR")
		}
		if missingCR {
			overrideLastSync = true
		}
	}

	// Then check if the next sync is due
	// Override if any of the tokens needs a refresh
	if cr.Status.LastSynced != 0 && !overrideLastSync {
		lastSync := time.Unix(cr.Status.LastSynced, 0)
		period, err := time.ParseDuration(cr.Spec.ResyncPeriod)
		if err != nil {
			metrics.IncreaseFailedConfigurationSyncsMetric()
			return v1.ResultFailed, errors2.Wrap(err, "error parsing operator resync period")
		}

		nextSync := lastSync.Add(period)
		if time.Now().Before(nextSync) {
			return v1.ResultSuccess, nil
		}
	}
	log.Info("operator resync window elapsed",
		"configured resync period", cr.Spec.ResyncPeriod)

	var selector labels.Selector

	if selector, err = buildConfigurationSelector(cr); err != nil {
		metrics.IncreaseFailedConfigurationSyncsMetric()
		return v1.ResultFailed, err
	}

	opts := &client.ListOptions{
		LabelSelector: selector,
	}

	// Get all configuration secret sets as well
	configSecretList := &corev1.SecretList{}

	if !cr.ExternalSyncDisabled() {
		err = r.client.List(ctx, configSecretList, opts)
		if err != nil {
			metrics.IncreaseFailedConfigurationSyncsMetric()
			return v1.ResultFailed, err
		}
	}
	// No configurations yet? Keep reconciling and don't wait for the resync period
	if len(configSecretList.Items) == 0 && !cr.ExternalSyncDisabled() {
		s.LastSynced = 0
		log.Info("no configurations found, resync window disabled awaiting initial config")
		return v1.ResultInProgress, nil
	}

	// Extract repository info
	log.Info("configurations found, resync initiated",
		"secret count", len(configSecretList.Items), "self contained", cr.ExternalSyncDisabled())
	repos := make(map[string]v1.RepositoryInfo)

	// pull all config repo indices from secrets first
	for _, configSecret := range configSecretList.Items {
		// take note if we hit any secrets with duplicate names and only keep the 1st one
		if _, found := repos[configSecret.Name]; found {
			log.Info("skipping duplicate configuration secret", "namespace", configSecret.Namespace,
				"name", configSecret.Name)
		} else {
			repos[configSecret.Name] = v1.RepositoryInfo{
				AccessToken: string(configSecret.Data[RemoteAccessToken]),
				Channel:     string(configSecret.Data[RemoteChannel]),
				Tag:         string(configSecret.Data[RemoteTag]),
				Repository:  string(configSecret.Data[RemoteRepository]),
				Source:      &configSecret,
			}
			// if Repository is not a Github repo, attempt to create resources deployment and service
			if !strings.HasPrefix(repos[configSecret.Name].Repository, "https://api.github.com") && !cr.ExternalSyncDisabled() {
				containerResources = true
				// build image repo tag
				image := repos[configSecret.Name].Repository + ":" + repos[configSecret.Name].Tag
				err = r.ReconcileResourcesDeployment(ctx, cr, image)
				if err != nil {
					metrics.IncreaseFailedConfigurationSyncsMetric()
					log.Error(err, "failed to create configuration resources deployment")
					return v1.ResultFailed, err
				}
				err = r.ReconcileResourcesService(ctx, cr)
				if err != nil {
					metrics.IncreaseFailedConfigurationSyncsMetric()
					log.Error(err, "failed to create configuration resources service")
					return v1.ResultFailed, err
				}

				// if running locally a resources route is required
				if utils.RunningLocally() {
					err = r.ReconcileResourcesRoute(ctx, cr, s)
					if err != nil {
						return v1.ResultFailed, err
					}
				}
			}
		}
	}

	// if deploying resources container ensure deployment and service are ready
	if containerResources {
		var status v1.ObservabilityStageStatus
		status, err = r.waitForResourcesDeployment(ctx, cr)
		if status != v1.ResultSuccess {
			return status, err
		}
	}

	// Collect index files
	var indexes []v1.RepositoryIndex
	for _, repoInfo := range repos {
		indexBytes, err := r.readIndexFile(&repoInfo, cr)
		if err != nil {
			metrics.IncreaseFailedConfigurationSyncsMetric()
			log.Error(err, "failed to fetch configuration repository index file")
			return v1.ResultFailed, err
		}

		var index v1.RepositoryIndex
		err = json.Unmarshal(indexBytes, &index)
		if err != nil {
			metrics.IncreaseFailedConfigurationSyncsMetric()
			log.Error(err, "failed to unmarshal configuration repository index")
			return v1.ResultFailed, err
		}

		baseUrl, err := r.getResourcesUrl(&repoInfo, cr)
		if err != nil {
			return v1.ResultFailed, err
		}

		index.BaseUrl = baseUrl.String()
		index.Tag = repoInfo.Tag
		index.AccessToken = repoInfo.AccessToken
		index.Source = repoInfo.Source
		indexes = append(indexes, index)
	}

	// Delete unrequested token secrets
	err = r.deleteUnrequestedCredentialSecrets(ctx, cr, indexes)
	if err != nil {
		metrics.IncreaseFailedConfigurationSyncsMetric()
		return v1.ResultFailed, err
	}

	for _, index := range indexes {
		err = token2.ReconcileObservatoria(r.logger, ctx, r.client, cr, &index)
		if err != nil {
			log.Error(err, "error configuring observatorium")
			continue
		}
		r.stampConfigSource(ctx, &index)
	}

	err = r.deleteUnrequestedTokenRefreshers(ctx, cr, indexes)
	if err != nil {
		metrics.IncreaseFailedConfigurationSyncsMetric()
		return v1.ResultFailed, errors2.Wrap(err, "error deleting unrequested token refreshers")
	}

	err = r.deleteUnrequestedNetworkPolicies(ctx, cr, indexes)
	if err != nil {
		metrics.IncreaseFailedConfigurationSyncsMetric()
		return v1.ResultFailed, errors2.Wrap(err, "error deleting unrequested network policies")
	}

	if !cr.ObservatoriumDisabled() {
		err = r.reconcileTokenRefresher(ctx, cr, indexes)
		if err != nil {
			metrics.IncreaseFailedConfigurationSyncsMetric()
			return v1.ResultFailed, errors2.Wrap(err, "error reconciling token refresher")
		}
	}

	// Alertmanager configuration
	// When external sync is disabled, allow to create secret
	if !cr.ExternalSyncDisabled() {
		overrideConfigSecret, _ := cr.HasAlertmanagerConfigSecret()

		// Only create the config secret if the user has not overridden it via CR
		if !overrideConfigSecret {
			err = r.reconcileAlertmanagerSecret(ctx, cr, indexes)
			if err != nil {
				metrics.IncreaseFailedConfigurationSyncsMetric()
				return v1.ResultFailed, errors2.Wrap(err, "error reconciling alertmanager secret")
			}
		}
	}

	// Prometheus additional scrape configs
	patterns, err := r.fetchFederationConfigs(cr, indexes)
	if err != nil {
		metrics.IncreaseFailedConfigurationSyncsMetric()
		return v1.ResultFailed, errors2.Wrap(err, "error fetching federation config")
	}
	err = r.createAdditionalScrapeConfigSecret(cr, ctx, patterns)
	if err != nil {
		metrics.IncreaseFailedConfigurationSyncsMetric()
		return v1.ResultFailed, err
	}
	//blackbox exporter
	hash, err := r.createBlackBoxConfig(cr, ctx)
	if err != nil {
		metrics.IncreaseFailedConfigurationSyncsMetric()
		return v1.ResultFailed, err
	}
	// Alertmanager CR
	err = r.reconcileAlertmanager(ctx, cr, indexes)
	if err != nil {
		metrics.IncreaseFailedConfigurationSyncsMetric()
		return v1.ResultFailed, errors2.Wrap(err, "error reconciling alertmanager")
	}

	// Prometheus CR
	err = r.reconcilePrometheus(ctx, cr, indexes, hash)
	if err != nil {
		metrics.IncreaseFailedConfigurationSyncsMetric()
		return v1.ResultFailed, errors2.Wrap(err, "error reconciling prometheus")
	}

	// Grafana CR
	if !cr.DescopedModeEnabled() {
		err = r.reconcileGrafanaCr(ctx, cr, indexes)
		if err != nil {
			metrics.IncreaseFailedConfigurationSyncsMetric()
			return v1.ResultFailed, errors2.Wrap(err, "error reconciling grafana")
		}
	}

	// Manage monitoring resources
	if !cr.ExternalSyncDisabled() {
		if !cr.DescopedModeEnabled() {
			dashboards := getUniqueDashboards(indexes)
			err = r.deleteUnrequestedDashboards(cr, ctx, dashboards)
			if err != nil {
				metrics.IncreaseFailedConfigurationSyncsMetric()
				return v1.ResultFailed, errors2.Wrap(err, "error deleting unrequested dashboards")
			}

			err = r.createRequestedDashboards(cr, ctx, dashboards)
			if err != nil {
				metrics.IncreaseFailedConfigurationSyncsMetric()
				return v1.ResultFailed, errors2.Wrap(err, "error creating requested dashboards")
			}
		}

		// Manage prometheus rules
		rules := getUniqueRules(indexes)
		err = r.deleteUnrequestedRules(cr, ctx, rules)
		if err != nil {
			metrics.IncreaseFailedConfigurationSyncsMetric()
			return v1.ResultFailed, errors2.Wrap(err, "error deleting unrequested prometheus rules")
		}

		err = r.createRequestedRules(cr, ctx, rules)
		if err != nil {
			metrics.IncreaseFailedConfigurationSyncsMetric()
			return v1.ResultFailed, errors2.Wrap(err, "error creating requested prometheus rules")
		}

		// Manage pod monitors
		monitors := getUniquePodMonitors(indexes)
		err = r.deleteUnrequestedPodMonitors(cr, ctx, monitors)
		if err != nil {
			metrics.IncreaseFailedConfigurationSyncsMetric()
			return v1.ResultFailed, errors2.Wrap(err, "error deleting unrequested pod monitors")
		}

		err = r.createRequestedPodMonitors(cr, ctx, monitors)
		if err != nil {
			metrics.IncreaseFailedConfigurationSyncsMetric()
			return v1.ResultFailed, errors2.Wrap(err, "error creating requested pod monitors")
		}
	} else {
		err = r.createDMSAlert(cr, ctx)
		if err != nil {
			metrics.IncreaseFailedConfigurationSyncsMetric()
			return v1.ResultFailed, errors2.Wrap(err, "error creating deadmansswitch alert")
		}
	}

	// Promtail instances
	// First cleanup any no longer requested instances
	err = r.deleteUnrequestedDaemonsets(ctx, cr, indexes)
	if err != nil {
		metrics.IncreaseFailedConfigurationSyncsMetric()
		return v1.ResultFailed, errors2.Wrap(err, "error deleting unrequested promtail daemon sets")
	}

	// Create requested promtail instances
	// There will be a dedicated instance for every index
	for _, index := range indexes {
		err = r.createPromtailDaemonsetFor(ctx, cr, &index)
		if err != nil {
			return v1.ResultFailed, errors2.Wrap(err, fmt.Sprintf("error creating promtail daemon set for %s", index.Id))
		}
	}

	// Next status: update timestamp
	if cr.ExternalSyncDisabled() {
		s.LastSynced = 0
	} else {
		s.LastSynced = time.Now().Unix()
	}
	metrics.IncreaseSuccessfulConfigurationSyncsMetric()
	return v1.ResultSuccess, nil
}

func buildConfigurationSelector(cr *v1.Observability) (labels.Selector, error) {
	selector := labels.NewSelector()

	if cr.Spec.ConfigurationSelector == nil {
		return selector, nil
	}

	if cr.Spec.ConfigurationSelector.MatchLabels != nil {
		matchLabels := labels.SelectorFromSet(cr.Spec.ConfigurationSelector.MatchLabels)
		requirements, _ := matchLabels.Requirements()

		for _, requirement := range requirements {
			selector = selector.Add(requirement)
		}
	}

	if cr.Spec.ConfigurationSelector.MatchExpressions != nil {
		for _, expr := range cr.Spec.ConfigurationSelector.MatchExpressions {
			requirement, err := labels.NewRequirement(expr.Key, selection.Operator(strings.ToLower(string(expr.Operator))), expr.Values)
			if err != nil {
				return labels.Nothing(), err
			}
			selector = selector.Add(*requirement)
		}
	}

	return selector, nil
}

func (r *Reconciler) deleteUnrequestedCredentialSecrets(ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex) error {
	list := corev1.SecretList{}
	selector := labels.SelectorFromSet(map[string]string{
		"managed-by": "observability-operator",
		"purpose":    "observatorium-token-secret",
	})
	opts := &client.ListOptions{
		Namespace:     cr.Namespace,
		LabelSelector: selector,
	}

	r.client.List(ctx, &list, opts)

	var expectedSecrets []string
	for _, index := range indexes {
		if index.Config == nil || index.Config.Observatoria == nil {
			continue
		}

		for _, observatorium := range index.Config.Observatoria {
			secretName := token2.GetObservatoriumTokenSecretName(&observatorium)
			expectedSecrets = append(expectedSecrets, secretName)
		}
	}

	secretExpected := func(name string) bool {
		for _, secretName := range expectedSecrets {
			if name == secretName {
				return true
			}
		}
		return false
	}

	for _, secret := range list.Items {
		if !secretExpected(secret.Name) {
			err := r.client.Delete(ctx, &secret)
			if err != nil {
				return errors2.Wrap(err, fmt.Sprintf("error deleting unrequested token secret %v", secret.Name))
			}
		}
	}

	return nil
}

func (r *Reconciler) getResourcesUrl(repo *v1.RepositoryInfo, cr *v1.Observability) (*url.URL, error) {
	// assume container resources if repo is not GitHub
	if !strings.HasPrefix(repo.Repository, "https://api.github.com") {
		if utils.RunningLocally() {
			if cr.Status.ResourcesRoute == "" {
				return nil, errors.NewBadRequest("resources route not ready yet")
			}
			return url.ParseRequestURI(fmt.Sprintf("http://%v/%v", cr.Status.ResourcesRoute, repo.Channel))
		} else {
			serviceUrl := fmt.Sprintf("http://%s.%s:8080", model.GetResourcesDefaultName(cr), cr.GetPrometheusOperatorNamespace())
			return url.ParseRequestURI(fmt.Sprintf("%s/%s", serviceUrl, repo.Channel))
		}
	}

	return url.ParseRequestURI(fmt.Sprintf("%s/%s", repo.Repository, repo.Channel))
}

func (r *Reconciler) readIndexFile(repo *v1.RepositoryInfo, cr *v1.Observability) ([]byte, error) {
	repoUrl, err := r.getResourcesUrl(repo, cr)
	if err != nil {
		return nil, err
	}

	repoUrl.Path = fmt.Sprintf("%v/%v", repoUrl.Path, "index.json")

	req, err := http.NewRequest(http.MethodGet, repoUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	if strings.Contains(repoUrl.Host, "github.com") {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", repo.AccessToken))
		req.Header.Set("Accept", "application/vnd.github.v3.raw")

		if repo.Tag != "" {
			q := req.URL.Query()
			q.Add("ref", repo.Tag)
			req.URL.RawQuery = q.Encode()
		}
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code when reading index file from %v: %v", req.URL.String(), resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (r *Reconciler) fetchResource(path string, tag string, token string) ([]byte, error) {
	resourceUrl, err := url.ParseRequestURI(path)
	if err != nil {
		return nil, errors2.Wrap(err, fmt.Sprintf("error parsing resource url: %s", path))
	}

	req, err := http.NewRequest(http.MethodGet, resourceUrl.String(), nil)
	if err != nil {
		return nil, errors2.Wrap(err, "error creating http request")
	}
	if token != "" && strings.Contains(path, "github.com") {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
		req.Header.Set("Accept", "application/vnd.github.v3.raw")

		if tag != "" {
			q := req.URL.Query()
			q.Add("ref", tag)
			req.URL.RawQuery = q.Encode()
		}
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, errors2.Wrap(err, fmt.Sprintf("error fetching resource from %s", path))
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code when resource from %v: %v", req.URL.String(), resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors2.Wrap(err, "error reading response")
	}

	return body, nil
}

func (r *Reconciler) checkForMissingCR(ctx context.Context, cr *v1.Observability) (bool, error) {
	var missingCR bool
	alertmanagerCR := model.GetAlertmanagerCr(cr)
	selector := client.ObjectKey{
		Namespace: cr.GetPrometheusOperatorNamespace(),
		Name:      model.GetDefaultNameAlertmanager(cr),
	}

	err := r.client.Get(ctx, selector, alertmanagerCR)
	if err != nil {
		if !errors.IsGone(err) {
			missingCR = true
		} else {
			return missingCR, err
		}
	}
	prometheusCR := model.GetPrometheus(cr)
	selector = client.ObjectKey{
		Namespace: cr.GetPrometheusOperatorNamespace(),
		Name:      model.GetDefaultNamePrometheus(cr),
	}

	err = r.client.Get(ctx, selector, prometheusCR)
	if err != nil {
		if !errors.IsGone(err) {
			missingCR = true
		} else {
			return missingCR, err
		}
	}

	if !cr.DescopedModeEnabled() {
		grafanaCR := model.GetGrafanaCr(cr)
		selector = client.ObjectKey{
			Namespace: cr.Namespace,
			Name:      model.GetDefaultNameGrafana(cr),
		}

		err = r.client.Get(ctx, selector, grafanaCR)
		if err != nil {
			if !errors.IsGone(err) {
				missingCR = true
			} else {
				return missingCR, err
			}
		}
	}
	return missingCR, nil
}

func (r *Reconciler) waitForResourcesService(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	service := model.GetResourcesService(cr)
	selector := client.ObjectKey{
		Namespace: cr.Namespace,
		Name:      model.GetResourcesDefaultName(cr),
	}
	err := r.client.Get(ctx, selector, service)
	if err != nil {
		if errors.IsNotFound(err) {
			return v1.ResultInProgress, nil
		}
		return v1.ResultFailed, err
	}

	if utils.IsServiceReady(service) {
		return v1.ResultSuccess, nil
	}
	return v1.ResultInProgress, nil
}

func (r *Reconciler) waitForResourcesDeployment(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	deployments := &appsv1.DeploymentList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := r.client.List(ctx, deployments, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	for _, deployment := range deployments.Items {
		if strings.HasPrefix(deployment.Name, model.GetResourcesDefaultName(cr)) {
			if deployment.Status.ReadyReplicas > 0 {
				return v1.ResultSuccess, nil
			}
		}
	}
	return v1.ResultInProgress, nil
}

func (r *Reconciler) GetOauth2ConfigSecret(ctx context.Context, cr *v1.Observability) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	selector := client.ObjectKey{
		Name:      RemoteWriteOIDCSecretName,
		Namespace: cr.Namespace,
	}

	err := r.client.Get(ctx, selector, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return secret, nil
}

func (r *Reconciler) GetRemoteWriteOauth2Config(ctx context.Context, cr *v1.Observability) (*prometheusv1.OAuth2, error) {
	secret, err := r.GetOauth2ConfigSecret(ctx, cr)
	if err != nil {
		return nil, err
	}

	if secret == nil {
		return nil, nil
	}

	tokenUrl, err := url.JoinPath(string(secret.Data["issuer_url"]), "/protocol/openid-connect/token")
	if err != nil {
		return nil, err
	}

	config := &prometheusv1.OAuth2{
		ClientID: prometheusv1.SecretOrConfigMap{
			Secret: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: RemoteWriteOIDCSecretName,
				},
				Key: "client_id",
			},
		},
		ClientSecret: corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: RemoteWriteOIDCSecretName,
			},
			Key: "client_secret",
		},
		TokenURL: tokenUrl,
	}

	return config, nil
}
