package configuration

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/model"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
	"github.com/jeremyary/observability-operator/controllers/token"
	"io/ioutil"
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
	RemoteRepository                = "repository"
	RemoteAccessToken               = "access_token"
	RemoteChannel                   = "channel"
	RemoteObservatoriumToken        = "observatorium_token"
	RemoteObservatoriumTokenExpires = "observatorium_token_expires"
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
	list := &v12.SecretList{}
	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"managed-by": "observability-operator",
		}),
		Namespace: cr.Namespace,
	}

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

	return v1.ResultSuccess, nil
}

func (r *Reconciler) refreshToken(ctx context.Context, cr *v1.Observability, index *v1.RepositoryIndex, from *v12.ConfigMap) error {
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

	oldToken := from.Data[RemoteObservatoriumToken]
	fetcher := token.GetTokenFetcher(index.Config.Prometheus.Observatorium, ctx, r.client)
	newToken, expires, err := fetcher.Fetch(cr, index.Config.Prometheus.Observatorium, oldToken)
	if err != nil {
		return err
	}

	// Write token secret
	_, err = controllerutil.CreateOrUpdate(ctx, r.client, secret, func() error {
		secret.StringData = map[string]string{
			"token": newToken,
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Write expiration info
	from.Data[RemoteObservatoriumTokenExpires] = strconv.FormatInt(expires, 10)
	err = r.client.Update(ctx, from)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	if cr.Spec.ConfigurationSelector == nil {
		r.logger.Info("warning: configuration label selector present, dynamic configuration will be skipped")
		return v1.ResultSuccess, nil
	}

	list := &v12.ConfigMapList{}
	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(cr.Spec.ConfigurationSelector.MatchLabels),
	}

	// Get all configuration sets
	err := r.client.List(ctx, list, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	needsRefresh := map[string]*v12.ConfigMap{}
	overrideLastSync := false

	// First check if any of the tokens has expired
	// If yes, we must force a sync
	for _, configMap := range list.Items {
		raw := configMap.Data[RemoteObservatoriumTokenExpires]
		repoChannel := configMap.Data[RemoteChannel]
		repoUrl := configMap.Data[RemoteRepository]
		repoChannelId := fmt.Sprintf("%s/%s", repoUrl, repoChannel)

		if raw == "" {
			needsRefresh[repoChannelId] = &configMap
			overrideLastSync = true
			continue
		}

		expires, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			r.logger.Error(err, "unknown expiration format, int64 expected")
			continue
		}

		if token.AuthTokenExpires(expires) {
			needsRefresh[repoChannelId] = &configMap
			overrideLastSync = true
			continue
		}
	}

	// Then check if the next sync is due
	// Override if any of the tokens needs a refresh
	if cr.Status.LastSynced != 0 && !overrideLastSync {
		lastSync := time.Unix(cr.Status.LastSynced, 0)
		period, err := time.ParseDuration(cr.Spec.ResyncPeriod)
		if err != nil {
			return v1.ResultFailed, err
		}

		nextSync := lastSync.Add(period)
		if time.Now().Before(nextSync) {
			return v1.ResultSuccess, nil
		}
	}

	// Extract repository info
	var repos []v1.RepositoryInfo
	for _, configMap := range list.Items {
		repoUrl := configMap.Data[RemoteRepository]
		_, err := url.Parse(repoUrl)
		if err != nil {
			r.logger.Error(err, "invalid repository url")
			continue
		}

		repos = append(repos, v1.RepositoryInfo{
			AccessToken: configMap.Data[RemoteAccessToken],
			Channel:     configMap.Data[RemoteChannel],
			Repository:  repoUrl,
		})
	}

	// Collect index files
	var indexes []v1.RepositoryIndex
	for _, repoInfo := range repos {
		indexBytes, err := r.readIndexFile(&repoInfo)
		if err != nil {
			r.logger.Error(err, "error reading repository index")
			continue
		}

		var index v1.RepositoryIndex
		err = json.Unmarshal(indexBytes, &index)
		if err != nil {
			r.logger.Error(err, "corrupt index file")
			continue
		}
		index.BaseUrl = fmt.Sprintf("%s/%s", repoInfo.Repository, repoInfo.Channel)
		index.AccessToken = repoInfo.AccessToken
		indexes = append(indexes, index)
	}

	// Refresh observatorium tokens
	for _, index := range indexes {
		if cm, ok := needsRefresh[index.BaseUrl]; ok && cm != nil {
			if index.Config != nil && index.Config.Prometheus != nil {
				err = r.refreshToken(ctx, cr, &index, cm)
				if err != nil {
					r.logger.Error(err, "unable to obtain observatorium token")
				}
			}
		}
	}

	// Prometheus additional scrape configs
	federationConfig, err := r.fetchFederationConfigs(indexes)
	if err != nil {
		return v1.ResultFailed, err
	}
	err = r.createAdditionalScrapeConfigSecret(cr, ctx, federationConfig)
	if err != nil {
		return v1.ResultFailed, err
	}

	// Prometheus CR
	err = r.reconcilePrometheus(ctx, cr)
	if err != nil {
		return v1.ResultFailed, err
	}

	// Manage dashboards
	dashboards := getUniqueDashboards(indexes)
	err = r.deleteUnrequestedDashboards(cr, ctx, dashboards)
	if err != nil {
		return v1.ResultFailed, err
	}

	err = r.createRequestedDashboards(cr, ctx, dashboards)
	if err != nil {
		return v1.ResultFailed, err
	}

	// Manage prometheus rules
	rules := getUniqueRules(indexes)
	err = r.deleteUnrequestedRules(cr, ctx, rules)
	if err != nil {
		return v1.ResultFailed, err
	}

	err = r.createRequestedRules(cr, ctx, rules)
	if err != nil {
		return v1.ResultFailed, err
	}

	// Next status: update timestamp
	s.LastSynced = time.Now().Unix()
	return v1.ResultSuccess, nil
}

func (r *Reconciler) readIndexFile(repo *v1.RepositoryInfo) ([]byte, error) {
	repoUrl := fmt.Sprintf("%s/%s/index.json", repo.Repository, repo.Channel)

	req, err := http.NewRequest(http.MethodGet, repoUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", repo.AccessToken))

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
	url, err := url.ParseRequestURI(path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
