package logging_installation

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	v14 "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	"github.com/redhat-developer/observability-operator/v4/controllers/model"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers"
	v12 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const NoInitConfigMapName = "observability-operator-no-init"

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

	managed, err := r.checkLoggingSubscriptionForLabel(ctx)
	if err != nil {
		return v1.ResultFailed, err
	}

	if managed {
		installedCsv, err := r.getInstalledLoggingCSVName(ctx)
		if installedCsv == "" || err != nil {
			return v1.ResultFailed, err
		}

		subscription := model.GetLoggingSubscription(cr)
		err = r.client.Delete(ctx, subscription)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		// Delete csv to uninstall
		csv := &v1alpha1.ClusterServiceVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      installedCsv,
				Namespace: "openshift-logging",
			},
		}
		err = r.client.Delete(ctx, csv)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		// Delete clusterLoggings
		cl := model.GetClusterLoggingCR()
		err = r.client.Delete(ctx, cl)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		// delete clusterLogForwarder
		clf := model.GetClusterLogForwarderCR()
		err = r.client.Delete(ctx, clf)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}
		return v1.ResultSuccess, nil
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	if cr.DescopedModeEnabled() {
		return v1.ResultSuccess, nil
	}

	// check for observability-operator-no-init configmap. If present check applied CR
	configMap := &corev1.ConfigMap{}
	selector := client.ObjectKey{
		Name:      NoInitConfigMapName,
		Namespace: cr.Namespace,
	}

	var selfContainedMode bool
	err := r.client.Get(ctx, selector, configMap)
	if err == nil {
		selfContainedMode = true
	}
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	// skip logging operator install if disableLogging is unset
	if selfContainedMode && cr.Spec.SelfContained != nil && cr.Spec.SelfContained.DisableLogging == nil {
		return v1.ResultSuccess, nil
	}

	// if logging is specifically disabled
	if cr.Spec.SelfContained.DisableLogging != nil && *cr.Spec.SelfContained.DisableLogging {
		return v1.ResultSuccess, nil
	}

	// if openshift-namespace is not present skip logging operator installation
	namespace := &corev1.Namespace{}
	selector = client.ObjectKey{
		Name: "openshift-logging",
	}

	err = r.client.Get(ctx, selector, namespace)
	if err != nil {
		if !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}
		return v1.ResultSuccess, nil
	}

	// Let's check if there is already a cluster-logging-operator
	// If the operator is already installed we want to leave it in place for now
	deployments := &v12.DeploymentList{}
	opts := &client.ListOptions{
		Namespace: "openshift-logging",
	}
	err = r.client.List(ctx, deployments, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	var foundLoggingOperator = false

	for _, deployment := range deployments.Items {
		if strings.HasPrefix(deployment.Name, "cluster-logging-operator") {
			foundLoggingOperator = true
		}
	}

	// If there was no logging-operator and we want to install the operator go ahead and set it up
	if !foundLoggingOperator {
		// logging subscription
		status, err := r.reconcileSubscription(ctx, cr)
		if status != v1.ResultSuccess {
			return status, err
		}

		status, err = r.waitForLoggingOperator(ctx, cr)
		if status != v1.ResultSuccess {
			return status, err
		}
	}

	// if we installed the logging operator then we want to set up our resources
	managed, err := r.checkLoggingSubscriptionForLabel(ctx)
	if err != nil {
		return v1.ResultFailed, err
	}

	if managed {
		status, err := r.createClusterLoggingCr(ctx, cr)
		if status != v1.ResultSuccess {
			return status, err
		}

		status, err = r.createClusterLogForwarderCr(ctx, cr)
		if status != v1.ResultSuccess {
			return status, err
		}
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {

	subscription := model.GetLoggingSubscription(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, subscription, func() error {
		subscription.Spec = &v1alpha1.SubscriptionSpec{
			CatalogSource:          "redhat-operators",
			CatalogSourceNamespace: "openshift-marketplace",
			Package:                "cluster-logging",
			Channel:                "stable",
			InstallPlanApproval:    v1alpha1.ApprovalAutomatic,
		}

		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) checkLoggingSubscriptionForLabel(ctx context.Context) (bool, error) {
	csv := &v1alpha1.SubscriptionList{}
	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{"app.kubernetes.io/managed-by": "observability-operator"}),
		Namespace:     "openshift-logging",
	}

	err := r.client.List(ctx, csv, opts)
	if err != nil {
		return false, err
	}

	if len(csv.Items) > 0 {
		return true, nil
	}

	return false, nil
}

func (r *Reconciler) getInstalledLoggingCSVName(ctx context.Context) (string, error) {
	csv := &v1alpha1.SubscriptionList{}
	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{"app.kubernetes.io/managed-by": "observability-operator"}),
		Namespace:     "openshift-logging",
	}

	err := r.client.List(ctx, csv, opts)
	if err != nil {
		return "", err
	}

	return csv.Items[0].Status.InstalledCSV, nil

}

func (r *Reconciler) waitForLoggingOperator(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	deployments := &v12.DeploymentList{}
	opts := &client.ListOptions{
		Namespace: "openshift-logging",
	}
	err := r.client.List(ctx, deployments, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	for _, deployment := range deployments.Items {
		if strings.HasPrefix(deployment.Name, "cluster-logging-operator") {
			if deployment.Status.ReadyReplicas > 0 {
				return v1.ResultSuccess, nil
			}
		}
	}
	return v1.ResultInProgress, nil
}

func (r *Reconciler) createClusterLoggingCr(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {

	// Is there any clusterLogging CR?
	opts := &client.ListOptions{
		Namespace: "openshift-logging",
	}

	list := &v14.ClusterLoggingList{}
	err := r.client.List(ctx, list, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	// Is there a clusterLogging CR with our label?
	labelOpts := &client.ListOptions{
		Namespace: "openshift-logging",
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"app.kubernetes.io/managed-by": "observability-operator",
		}),
	}
	labelList := &v14.ClusterLoggingList{}
	err = r.client.List(ctx, labelList, labelOpts)
	if err != nil {
		return v1.ResultFailed, err
	}

	if len(list.Items) == 0 || len(labelList.Items) > 0 {
		// There's no ClusterLogging or one that we manage
		clCr := model.GetClusterLoggingCR()
		_, err = controllerutil.CreateOrUpdate(ctx, r.client, clCr, func() error {
			return nil
		})

		if err != nil {
			return v1.ResultFailed, err
		}
		return v1.ResultSuccess, nil
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) createClusterLogForwarderCr(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {

	// Is there any clusterLogForwarder CR?
	opts := &client.ListOptions{
		Namespace: "openshift-logging",
	}

	list := &v14.ClusterLogForwarderList{}
	err := r.client.List(ctx, list, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	// Is there a clusterLogForwarder CR with our label?
	labelOpts := &client.ListOptions{
		Namespace: "openshift-logging",
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"app.kubernetes.io/managed-by": "observability-operator",
		}),
	}
	labelList := &v14.ClusterLogForwarderList{}
	err = r.client.List(ctx, labelList, labelOpts)
	if err != nil {
		return v1.ResultFailed, err
	}

	if len(list.Items) == 0 || len(labelList.Items) > 0 {
		// There's no ClusterLogforwarder or one that we manage
		// Add the operator namespaces
		clusterLogForwarder := model.GetClusterLogForwarderCR()
		newPipeline := model.GetClusterLogForwarderPipeline()
		clusterLogForwarder.Spec.Pipelines = append(clusterLogForwarder.Spec.Pipelines, *newPipeline)

		// Check for any namespaces that contain a managedkafka resource
		// and add them to the clusterLogForwarder
		opts := &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app.kubernetes.io/managed-by": "kas-fleetshard-operator",
			}),
		}
		list := &corev1.NamespaceList{}
		err := r.client.List(ctx, list, opts)
		if err != nil {
			return v1.ResultFailed, err
		}

		namespaces := []string{}

		if len(list.Items) > 0 {

			for _, namespace := range list.Items {
				namespaces = append(namespaces, namespace.Name)
			}

			kafkaInput := v14.InputSpec{
				Name: "kafka-log-resources",
				Application: &v14.Application{
					Namespaces: namespaces,
					Selector:   nil,
				},
				Infrastructure: nil,
				Audit:          nil,
			}

			clusterLogForwarder.Spec.Inputs = append(clusterLogForwarder.Spec.Inputs, kafkaInput)
			newPipeline := model.GetClusterLogForwarderPipeline()
			newPipeline.InputRefs = append(newPipeline.InputRefs, "kafka-log-resources")
			clusterLogForwarder.Spec.Pipelines = []v14.PipelineSpec{*newPipeline}
		}

		_, err = controllerutil.CreateOrUpdate(ctx, r.client, clusterLogForwarder, func() error {
			return nil
		})
		if err != nil {
			return v1.ResultFailed, err
		}

		if err != nil {
			return v1.ResultFailed, err
		}
	}

	return v1.ResultSuccess, nil
}
