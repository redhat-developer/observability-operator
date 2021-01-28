package configuration

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
	"github.com/jeremyary/observability-operator/controllers/token"
	"io/ioutil"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"net/http"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (r *Reconciler) refreshToken(ctx context.Context, cr *v1.Observability, target *v12.ConfigMap, index *v1.ObservatoriumIndex) error {
	if index == nil {
		return nil
	}

	oldToken := target.Data[RemoteObservatoriumToken]
	fetcher := token.GetTokenFetcher(index, ctx, r.client)
	newToken, expires, err := fetcher.Fetch(cr, index, oldToken)
	if err != nil {
		return err
	}

	target.Data[RemoteObservatoriumToken] = newToken
	target.Data[RemoteObservatoriumTokenExpires] = strconv.FormatInt(expires, 10)
	err = r.client.Update(ctx, target)
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
		oldToken := configMap.Data[RemoteObservatoriumToken]
		repoChannel := configMap.Data[RemoteChannel]
		repoUrl := configMap.Data[RemoteRepository]
		repoChannelId := fmt.Sprintf("%s/%s", repoUrl, repoChannel)

		if oldToken == "" || raw == "" {
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
				err = r.refreshToken(ctx, cr, cm, index.Config.Prometheus.Observatorium)
				if err != nil {
					r.logger.Error(err, "unable to obtain observatorium token")
				}
			}
		}
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

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
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
