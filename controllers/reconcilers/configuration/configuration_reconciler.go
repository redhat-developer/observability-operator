package configuration

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/api/v1"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/controllers/model"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/controllers/reconcilers"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/controllers/token"
	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	"github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	errors2 "github.com/pkg/errors"
	"io/ioutil"
	v13 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"net/http"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
	"time"
)

const (
	RemoteRepository            = "repository"
	RemoteAccessToken           = "access_token"
	RemoteChannel               = "channel"
	PrometheusRuleIdentifierKey = "observability"
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

	list := &v12.SecretList{}
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

	configMapList := &v12.ConfigMapList{}
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

	// Delete Promtail daemonsets
	daemonsetList := &v13.DaemonSetList{}
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

	return v1.ResultSuccess, nil
}

func (r *Reconciler) refreshToken(ctx context.Context, cr *v1.Observability, index *v1.RepositoryIndex, lifetimeStorage *v12.ConfigMap) error {
	log := r.logger.WithValues("observability", cr.Name)
	if index == nil || index.Config == nil || index.Config.Prometheus == nil || index.Config.Prometheus.Observatorium == nil {
		return nil
	}

	secretName := fmt.Sprintf("%s-%s", index.Id, "observatorium-credentials")
	secret := model.GetTokenSecret(cr, secretName)
	selector := client.ObjectKey{
		Namespace: cr.Namespace,
		Name:      secretName,
	}

	err := r.client.Get(ctx, selector, secret)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	oldToken := secret.StringData["token"]
	fetcher := token.GetTokenFetcher(index.Config.Prometheus.Observatorium, ctx, r.client)
	newToken, expires, err := fetcher.Fetch(cr, index.Config.Prometheus.Observatorium, oldToken)
	if err != nil {
		log.Error(err, "failed to fetch auth token")
		return err
	} else {
		log.Info("new observatorium token expiry", "epoch", expires)
	}

	// Write token secret
	_, err = controllerutil.CreateOrUpdate(ctx, r.client, secret, func() error {
		secret.Labels = map[string]string{
			"managed-by": "observability-operator",
			"purpose":    "observatorium-token-secret",
		}
		secret.StringData = map[string]string{
			"token": newToken,
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Write expiration info
	if lifetimeStorage.Data == nil {
		lifetimeStorage.Data = map[string]string{}
	}
	lifetimeStorage.Data[index.Id] = strconv.FormatInt(expires, 10)
	err = r.client.Update(ctx, lifetimeStorage)
	if err != nil {
		return err
	}

	if index.MapSource != nil {
		// Update source configmap
		if index.MapSource.Annotations == nil {
			index.MapSource.Annotations = map[string]string{}
		}
		index.MapSource.Annotations["observability-operator/status"] = "accepted"
		err = r.client.Update(ctx, index.MapSource)
		if err != nil {
			return err
		}
	} else if index.SecretSource != nil {
		// Update source secret
		if index.SecretSource.Annotations == nil {
			index.SecretSource.Annotations = map[string]string{}
		}
		index.SecretSource.Annotations["observability-operator/status"] = "accepted"
		err = r.client.Update(ctx, index.SecretSource)
		if err != nil {
			return err
		}
	} else {
		log.Info("skipping update to repositoryIndex source, unknown source type")
	}

	return nil
}

func (r *Reconciler) getTokenLifetimeStorage(ctx context.Context, cr *v1.Observability) (*v12.ConfigMap, error) {
	storage := model.GetPrometheusAuthTokenLifetimes(cr)
	selector := client.ObjectKey{
		Namespace: cr.Namespace,
		Name:      storage.Name,
	}

	err := r.client.Get(ctx, selector, storage)
	if err != nil {
		return nil, err
	}

	return storage, err
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	log := r.logger.WithValues("observability", cr.Name)
	if cr.Spec.ConfigurationSelector == nil {
		log.Info("warning: configuration label selector present, dynamic configuration will be skipped")
		return v1.ResultSuccess, nil
	}

	tokensValid := map[string]bool{}
	overrideLastSync := false

	tokenLifetimes, err := r.getTokenLifetimeStorage(ctx, cr)
	if err != nil {
		return v1.ResultFailed, errors2.Wrap(err, "error finding or creating token lifetime cache")
	}

	// First check if any of the tokens have expired
	for id, lifetime := range tokenLifetimes.Data {
		if lifetime == "" {
			tokensValid[id] = false
			overrideLastSync = true
			log.Info("token expired, overriding lastSync for re-fetch")
			continue
		}

		expires, err := strconv.ParseInt(lifetime, 10, 64)
		if err != nil {
			log.Error(err, "unknown expiration format, int64 expected")
			continue
		}

		if token.AuthTokenExpires(expires) {
			tokensValid[id] = false
			overrideLastSync = true
			log.Info("auth token expired, overriding lastSync for re-fetch")
			continue
		}

		tokensValid[id] = true
	}

	// Then check if the next sync is due
	// Override if any of the tokens needs a refresh
	if cr.Status.LastSynced != 0 && !overrideLastSync {
		lastSync := time.Unix(cr.Status.LastSynced, 0)
		period, err := time.ParseDuration(cr.Spec.ResyncPeriod)
		if err != nil {
			return v1.ResultFailed, errors2.Wrap(err, "error parsing operator resync period")
		}

		nextSync := lastSync.Add(period)
		if time.Now().Before(nextSync) {
			return v1.ResultSuccess, nil
		}
	}
	log.Info("operator resync window elapsed, proceeding with re-fetch",
		"configured resync period", cr.Spec.ResyncPeriod)

	configMapList := &v12.ConfigMapList{}
	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(cr.Spec.ConfigurationSelector.MatchLabels),
	}

	// Get all configuration configMap sets
	err = r.client.List(ctx, configMapList, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	configSecretList := &v12.SecretList{}
	err = r.client.List(ctx, configSecretList, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	// No configurations yet? Keep reconciling and don't wait for the resync period
	if len(configMapList.Items) == 0 && len(configSecretList.Items) == 0 {
		s.LastSynced = 0
		log.Info("no configurations found, resync window disabled awaiting initial config")
		return v1.ResultInProgress, nil
	}

	// Extract repository info
	log.Info("configurations found, resync initiated", "map count", len(configMapList.Items),
		"secret count", len(configSecretList.Items))
	var repos []v1.RepositoryInfo
	for _, configMap := range configMapList.Items {
		repoUrl := configMap.Data[RemoteRepository]
		_, err := url.Parse(repoUrl)
		if err != nil {
			log.Error(err, "failed to resync configuration from map, invalid repository url specified")
			continue
		}

		repos = append(repos, v1.RepositoryInfo{
			AccessToken: configMap.Data[RemoteAccessToken],
			Channel:     configMap.Data[RemoteChannel],
			Repository:  repoUrl,
			MapSource:   &configMap,
		})
	}

	for _, configSecret := range configSecretList.Items {
		repoUrl := string(configSecret.Data[RemoteRepository])
		_, err := url.Parse(repoUrl)
		if err != nil {
			log.Error(err, "failed to resync configuration from secret, invalid repository url specified")
			continue
		}

		repos = append(repos, v1.RepositoryInfo{
			AccessToken:  string(configSecret.Data[RemoteAccessToken]),
			Channel:      string(configSecret.Data[RemoteChannel]),
			Repository:   repoUrl,
			SecretSource: &configSecret,
		})
	}

	// Collect index files
	var indexes []v1.RepositoryIndex
	for _, repoInfo := range repos {
		indexBytes, err := r.readIndexFile(&repoInfo)
		if err != nil {
			log.Error(err, "failed to fetch configuration repository index file")
			return v1.ResultFailed, err
		}

		var index v1.RepositoryIndex
		err = json.Unmarshal(indexBytes, &index)
		if err != nil {
			log.Error(err, "failed to unmarshal configuration repository index")
			return v1.ResultFailed, err
		}
		index.BaseUrl = fmt.Sprintf("%s/%s", repoInfo.Repository, repoInfo.Channel)
		index.AccessToken = repoInfo.AccessToken
		index.MapSource = repoInfo.MapSource
		index.SecretSource = repoInfo.SecretSource
		indexes = append(indexes, index)
	}

	// Refresh observatorium tokens
	for _, index := range indexes {
		if val, ok := tokensValid[index.Id]; !ok || !val {
			if index.Config != nil && index.Config.Prometheus != nil {
				log.Info("observatorium token missing or expired, refreshing")
				err = r.refreshToken(ctx, cr, &index, tokenLifetimes)
				if err != nil {
					log.Error(err, "failed to refresh observatorium token")
				}
			}
		}
	}

	// Alertmanager configuration
	err = r.reconcileAlertmanagerSecret(ctx, cr, indexes)
	if err != nil {
		return v1.ResultFailed, errors2.Wrap(err, "error reconciling alertmanager secret")
	}

	// Prometheus additional scrape configs
	patterns, err := r.fetchFederationConfigs(indexes)
	if err != nil {
		return v1.ResultFailed, errors2.Wrap(err, "error fetching federation config")
	}
	err = r.createAdditionalScrapeConfigSecret(cr, ctx, patterns)
	if err != nil {
		return v1.ResultFailed, err
	}

	// Alertmanager CR
	err = r.reconcileAlertmanager(ctx, cr)
	if err != nil {
		return v1.ResultFailed, errors2.Wrap(err, "error reconciling alertmanager")
	}

	// Prometheus CR
	err = r.reconcilePrometheus(ctx, cr, indexes)
	if err != nil {
		return v1.ResultFailed, errors2.Wrap(err, "error reconciling prometheus")
	}

	// Manage dashboards
	dashboards := getUniqueDashboards(indexes)
	err = r.deleteUnrequestedDashboards(cr, ctx, dashboards)
	if err != nil {
		return v1.ResultFailed, errors2.Wrap(err, "error deleting unrequested dashboards")
	}

	err = r.createRequestedDashboards(cr, ctx, dashboards)
	if err != nil {
		return v1.ResultFailed, errors2.Wrap(err, "error creating requested dashboards")
	}

	// Manage prometheus rules
	rules := getUniqueRules(indexes)
	err = r.deleteUnrequestedRules(cr, ctx, rules)
	if err != nil {
		return v1.ResultFailed, errors2.Wrap(err, "error deleting unrequested prometheus rules")
	}

	err = r.createRequestedRules(cr, ctx, rules)
	if err != nil {
		return v1.ResultFailed, errors2.Wrap(err, "error creating requested prometheus rules")
	}

	// Manage pod monitors
	monitors := getUniquePodMonitors(indexes)
	err = r.deleteUnrequestedPodMonitors(cr, ctx, monitors)
	if err != nil {
		return v1.ResultFailed, errors2.Wrap(err, "error deleting unrequested pod monitors")
	}

	err = r.createRequestedPodMonitors(cr, ctx, monitors)
	if err != nil {
		return v1.ResultFailed, errors2.Wrap(err, "error creating requested pod monitors")
	}

	// Promtai instances
	// First cleanup any no longer requested instances
	err = r.deleteUnrequestedDaemonsets(ctx, cr, indexes)
	if err != nil {
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
	s.LastSynced = time.Now().Unix()
	return v1.ResultSuccess, nil
}

func (r *Reconciler) readIndexFile(repo *v1.RepositoryInfo) ([]byte, error) {
	repoUrl, err := url.ParseRequestURI(fmt.Sprintf("%s/%s/index.json", repo.Repository, repo.Channel))
	if err != nil {
		return nil, err
	}

	if repo.AccessToken == "" {
		return nil, fmt.Errorf("repository ConfigMap missing required AccessToken")
	}

	req, err := http.NewRequest(http.MethodGet, repoUrl.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", repo.AccessToken))
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (r *Reconciler) fetchResource(path string, token string) ([]byte, error) {
	resourceUrl, err := url.ParseRequestURI(path)
	if err != nil {
		return nil, errors2.Wrap(err, fmt.Sprintf("error parsing resource url: %s", path))
	}

	if token == "" {
		return nil, fmt.Errorf("repository ConfigMap missing required AccessToken")
	}

	req, err := http.NewRequest(http.MethodGet, resourceUrl.String(), nil)
	if err != nil {
		return nil, errors2.Wrap(err, "error creating http request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, errors2.Wrap(err, fmt.Sprintf("error fetching resource from %s", path))
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors2.Wrap(err, "error reading response")
	}

	return body, nil
}
