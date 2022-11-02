package grafana_installation

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	coreosv1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	"github.com/redhat-developer/observability-operator/v3/controllers/model"
	"github.com/redhat-developer/observability-operator/v3/controllers/reconcilers"
	"github.com/redhat-developer/observability-operator/v3/controllers/utils"
	v12 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const GrafanaOperatorDefaultVersion = "grafana-operator.v4.7.0"

type Reconciler struct {
	client client.Client
	logger logr.Logger
}

func NewReconciler(client client.Client, logger logr.Logger) reconcilers.ObservabilityReconciler {
	return &Reconciler{
		client: client,
		logger: logger,
	}
}

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	if cr.DescopedModeEnabled() {
		return v1.ResultSuccess, nil
	}

	source := model.GetGrafanaCatalogSource(cr)
	err := r.client.Delete(ctx, source)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	subscription := model.GetGrafanaSubscription(cr)
	err = r.client.Delete(ctx, subscription)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	operatorgroup := model.GetGrafanaOperatorGroup(cr)
	err = r.client.Delete(ctx, operatorgroup)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	// We have to remove the grafana operator deployment manually
	deployments := &v12.DeploymentList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err = r.client.List(ctx, deployments, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	for _, deployment := range deployments.Items {
		if deployment.Name == "grafana-operator" {
			err = r.client.Delete(ctx, &deployment)
			if err != nil && !errors.IsNotFound(err) {
				return v1.ResultFailed, err
			}
		}
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	if cr.DescopedModeEnabled() {
		return v1.ResultSuccess, nil
	}

	// Grafana catalog source
	status, err := r.reconcileCatalogSource(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// Grafana subscription
	status, err = r.reconcileSubscription(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// Observability operator group
	status, err = r.reconcileOperatorgroup(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.waitForGrafanaOperator(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileCatalogSource(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	source := &v1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-operator-catalog-source",
			Namespace: cr.Namespace,
		},
	}
	selector := client.ObjectKey{
		Namespace: source.Namespace,
		Name:      source.Name,
	}

	//look for catalogSource for old Grafana Operator index. If found migrate to community image
	err := r.client.Get(ctx, selector, source)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}
	if err == nil {
		if source.Spec.Image == "quay.io/rhoas/grafana-operator-index:v3.10.5" {
			err := r.removeUnusedGrafanaOperatorIndexResources(ctx, source, cr)
			if err != nil {
				return v1.ResultFailed, err
			}
			return v1.ResultSuccess, nil
		}
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	subscription := model.GetGrafanaSubscription(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, subscription, func() error {
		subscription.Spec = &v1alpha1.SubscriptionSpec{
			CatalogSource:          "community-operators",
			CatalogSourceNamespace: "openshift-marketplace",
			Package:                "grafana-operator",
			Channel:                "v4",
			InstallPlanApproval:    v1alpha1.ApprovalManual,
			Config:                 &v1alpha1.SubscriptionConfig{Resources: model.GetGrafanaOperatorResourceRequirement(cr)},
			StartingCSV:            GrafanaOperatorDefaultVersion,
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	err = r.approveGrafanaOperatorInstallPlan(ctx, cr)
	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileOperatorgroup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	exists, err := utils.HasOperatorGroupForNamespace(ctx, r.client, cr.Namespace)
	if err != nil {
		return v1.ResultFailed, err
	}

	if exists {
		return v1.ResultSuccess, nil
	}

	operatorgroup := model.GetGrafanaOperatorGroup(cr)

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, operatorgroup, func() error {
		operatorgroup.Spec = coreosv1.OperatorGroupSpec{
			TargetNamespaces: []string{cr.Namespace},
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) waitForGrafanaOperator(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	// We have to remove the prometheus operator deployment manually
	deployments := &v12.DeploymentList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := r.client.List(ctx, deployments, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	for _, deployment := range deployments.Items {
		if strings.HasPrefix(deployment.Name, "grafana-operator") {
			if deployment.Status.ReadyReplicas > 0 {
				return v1.ResultSuccess, nil
			}
		}
	}
	return v1.ResultInProgress, nil
}

func (r *Reconciler) removeUnusedGrafanaOperatorIndexResources(ctx context.Context, source *v1alpha1.CatalogSource, cr *v1.Observability) error {
	// Delete subscription
	subscription := &v1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-subscription",
			Namespace: cr.Namespace,
		},
	}
	err := r.client.Delete(ctx, subscription)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	// Delete catalog source
	err = r.client.Delete(ctx, source)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	// Delete csv to uninstall
	csv := &v1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-operator.v3.10.5",
			Namespace: cr.Namespace,
		},
	}
	err = r.client.Delete(ctx, csv)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}

func (r *Reconciler) approveGrafanaOperatorInstallPlan(ctx context.Context, cr *v1.Observability) error {
	plans := &v1alpha1.InstallPlanList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}
	err := r.client.List(ctx, plans, opts)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	for _, plan := range plans.Items {
		if plan.Spec.ClusterServiceVersionNames[0] == GrafanaOperatorDefaultVersion && !plan.Spec.Approved {
			plan.Spec.Approved = true
			err := r.client.Update(ctx, &plan)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
