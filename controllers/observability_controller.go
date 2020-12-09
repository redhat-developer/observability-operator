package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
	"github.com/jeremyary/observability-operator/controllers/reconcilers/alertmanager_installation"
	"github.com/jeremyary/observability-operator/controllers/reconcilers/csv"
	"github.com/jeremyary/observability-operator/controllers/reconcilers/grafana_configuration"
	"github.com/jeremyary/observability-operator/controllers/reconcilers/grafana_installation"
	"github.com/jeremyary/observability-operator/controllers/reconcilers/prometheus_configuration"
	"github.com/jeremyary/observability-operator/controllers/reconcilers/prometheus_installation"
	"github.com/jeremyary/observability-operator/controllers/reconcilers/prometheus_rules"
	"github.com/jeremyary/observability-operator/controllers/reconcilers/promtail_installation"
	"github.com/jeremyary/observability-operator/controllers/reconcilers/token"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	apiv1 "github.com/jeremyary/observability-operator/api/v1"
)

const (
	RequeueDelaySuccess    = 10 * time.Second
	RequeueDelayError      = 5 * time.Second
	ObservabilityFinalizer = "observability-cleanup"
)

// ObservabilityReconciler reconciles a Observability object
type ObservabilityReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=observability.redhat.com,resources=observabilities,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=observability.redhat.com,resources=observabilities/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=corev1,resources=configmaps,verbs=get;list;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=podmonitors;alertmanagers;prometheuses;prometheuses/finalizers;alertmanagers/finalizers;servicemonitors;prometheusrules;thanosrulers;thanosrulers/finalizers,verbs=get;list;create;update;patch;delete;watch
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,resourceNames=privileged,verbs=use
// +kubebuilder:rbac:groups=integreatly.org,resources=grafanas;grafanadashboards;grafanadatasources,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:urls=/metrics,verbs=get
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
// +kubebuilder:rbac:groups=authentication.k8s.io,resources=tokenreviews,verbs=create
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:groups=apps,resources=deployments;daemonsets;statefulsets,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions;operatorgroups;clusterserviceversions,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:groups=,resources=namespaces;pods;nodes;nodes/proxy,verbs=get;list;watch
// +kubebuilder:rbac:groups=,resources=secrets;serviceaccounts;configmaps;endpoints;services;nodes/proxy,verbs=get;list;create;update;delete;watch

func (r *ObservabilityReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("observability", req.NamespacedName)

	// fetch Observability instance
	obs := &apiv1.Observability{}
	err := r.Get(ctx, req.NamespacedName, obs)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// CR deleted since request queued, child objects getting GC'd, no requeue
			log.Info("Observability CR not found, has been deleted")
			return ctrl.Result{}, nil
		}
		// error fetching observability instance, requeue and try again
		log.Error(err, "Error in Get of Observability CR")
		return ctrl.Result{}, err
	}

	// Add a cleanup finalizer if not already present
	if obs.DeletionTimestamp == nil && len(obs.Finalizers) == 0 {
		obs.Finalizers = append(obs.Finalizers, ObservabilityFinalizer)
		err = r.Update(ctx, obs)
		return ctrl.Result{}, err
	}

	var finished = true

	var stages []apiv1.ObservabilityStageName
	if obs.DeletionTimestamp == nil {
		stages = r.getInstallationStages()
	} else {
		stages = r.getCleanupStages()
	}

	nextStatus := obs.Status.DeepCopy()

	for _, stage := range stages {
		nextStatus.Stage = stage

		reconciler := r.getReconcilerForStage(stage)
		if reconciler != nil {
			var status apiv1.ObservabilityStageStatus
			var err error

			if obs.DeletionTimestamp == nil {
				status, err = reconciler.Reconcile(ctx, obs, nextStatus)
			} else {
				status, err = reconciler.Cleanup(ctx, obs)
			}

			if err != nil {
				r.Log.Error(err, fmt.Sprintf("reconciler error in stage %v", stage))
				nextStatus.LastMessage = err.Error()
			}

			nextStatus.StageStatus = status

			// If a stage is not complete, do not continue with the next
			if status != apiv1.ResultSuccess {
				finished = false
				break
			}
		}
	}

	// Ready for deletion?
	// Only remove the finalizer when all stages were successful
	if obs.DeletionTimestamp != nil && finished {
		obs.Finalizers = []string{}
		err = r.Update(ctx, obs)
		return ctrl.Result{}, err
	}

	return r.updateStatus(obs, nextStatus)
}

func (r *ObservabilityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.Observability{}).
		Complete(r)
}

func (r *ObservabilityReconciler) getInstallationStages() []apiv1.ObservabilityStageName {
	return []apiv1.ObservabilityStageName{
		apiv1.TokenRequest,
		apiv1.PrometheusInstallation,
		apiv1.PrometheusConfiguration,
		apiv1.PrometheusRules,
		apiv1.GrafanaInstallation,
		apiv1.GrafanaConfiguration,
		apiv1.PromtailInstallation,
		apiv1.AlertmanagerInstallation,
	}
}

func (r *ObservabilityReconciler) getCleanupStages() []apiv1.ObservabilityStageName {
	return []apiv1.ObservabilityStageName{
		apiv1.PrometheusRules,
		apiv1.PrometheusConfiguration,
		apiv1.GrafanaConfiguration,
		apiv1.PrometheusInstallation,
		apiv1.GrafanaInstallation,
		apiv1.PromtailInstallation,
		apiv1.AlertmanagerInstallation,
		apiv1.TokenRequest,
		apiv1.CsvRemoval,
	}
}

func (r *ObservabilityReconciler) updateStatus(cr *apiv1.Observability, nextStatus *apiv1.ObservabilityStatus) (ctrl.Result, error) {
	if !reflect.DeepEqual(&cr.Status, nextStatus) {
		nextStatus.DeepCopyInto(&cr.Status)
		err := r.Client.Status().Update(context.Background(), cr)
		if err != nil {
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: RequeueDelayError,
			}, err
		}
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: RequeueDelaySuccess,
	}, nil
}

func (r *ObservabilityReconciler) getReconcilerForStage(stage apiv1.ObservabilityStageName) reconcilers.ObservabilityReconciler {
	switch stage {
	case apiv1.PrometheusInstallation:
		return prometheus_installation.NewReconciler(r.Client, r.Log, r.Scheme)

	case apiv1.PrometheusConfiguration:
		return prometheus_configuration.NewReconciler(r.Client, r.Log)

	case apiv1.PrometheusRules:
		return prometheus_rules.NewReconciler(r.Client, r.Log)

	case apiv1.GrafanaInstallation:
		return grafana_installation.NewReconciler(r.Client, r.Log)

	case apiv1.GrafanaConfiguration:
		return grafana_configuration.NewReconciler(r.Client, r.Log)

	case apiv1.CsvRemoval:
		return csv.NewReconciler(r.Client, r.Log)

	case apiv1.TokenRequest:
		return token.NewReconciler(r.Client, r.Log)

	case apiv1.PromtailInstallation:
		return promtail_installation.NewReconciler(r.Client, r.Log)

	case apiv1.AlertmanagerInstallation:
		return alertmanager_installation.NewReconciler(r.Client, r.Log)

	default:
		return nil
	}
}
