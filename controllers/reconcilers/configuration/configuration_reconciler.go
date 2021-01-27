package configuration

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	v1 "github.com/jeremyary/observability-operator/api/v1"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
	"io/ioutil"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"net/http"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RemoteRepository  = "repository"
	RemoteAccessToken = "access_token"
	RemoteChannel     = "channel"
)

type RepositoryInfo struct {
	Repository  string
	Channel     string
	AccessToken string
}

type GrafanaIndex struct {
	Dashboards []string `json:"dashboards"`
}

type RepositoryConfig struct {
	Grafana *GrafanaIndex `json:"grafana"`
}

type RepositoryIndex struct {
	Config *RepositoryConfig `json:"config"`
}

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

	// Extract repository info
	var repos []RepositoryInfo
	for _, configMap := range list.Items {
		repoUrl := configMap.Data[RemoteRepository]
		_, err := url.Parse(repoUrl)
		if err != nil {
			r.logger.Error(err, "invalid repository url")
			continue
		}

		repos = append(repos, RepositoryInfo{
			AccessToken: configMap.Data[RemoteAccessToken],
			Channel:     configMap.Data[RemoteChannel],
			Repository:  repoUrl,
		})
	}

	// Collect index files
	var indexes []RepositoryIndex
	for _, repoInfo := range repos {
		indexBytes, err := r.readIndexFile(&repoInfo)
		if err != nil {
			r.logger.Error(err, "error reading repository index")
			continue
		}

		var index RepositoryIndex
		err = json.Unmarshal(indexBytes, &index)
		if err != nil {
			r.logger.Error(err, "corrupt index file")
			continue
		}

		indexes = append(indexes, index)
	}

	// Manage dashboards
	dashboards := getUniqueDashboards(indexes)
	deleteUnrequestedDashboards(cr, ctx, r.client, dashboards)

	return v1.ResultSuccess, nil
}

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	return v1.ResultSuccess, nil
}

func (r *Reconciler) readIndexFile(repo *RepositoryInfo) ([]byte, error) {
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
