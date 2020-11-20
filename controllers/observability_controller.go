package controllers

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/jeremyary/observability-operator/controllers/reconcilers"
	"github.com/jeremyary/observability-operator/controllers/reconcilers/grafana_configuration"
	"github.com/jeremyary/observability-operator/controllers/reconcilers/grafana_installation"
	"github.com/jeremyary/observability-operator/controllers/reconcilers/prometheus_configuration"
	"github.com/jeremyary/observability-operator/controllers/reconcilers/prometheus_installation"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	apiv1 "github.com/jeremyary/observability-operator/api/v1"
)

const (
	RequeueDelaySuccess = 10 * time.Second
	RequeueDelayError   = 5 * time.Second
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
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=podmonitors,verbs=get;list;create;update;patch;delete

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

	var lastStage apiv1.ObservabilityStageName
	var lastStatus apiv1.ObservabilityStageStatus
	var lastMessage = ""

	for _, stage := range r.getStages() {
		lastStage = stage

		reconciler := r.getReconcilerForStage(stage)
		if reconciler != nil {
			status, err := reconciler.Reconcile(ctx, obs)
			if err != nil {
				lastMessage = err.Error()
			}

			lastStatus = status

			// If a stage is not complete, do not continue with the next
			if status != apiv1.ResultSuccess {
				break
			}
		}
	}

	return r.updateStatus(obs, lastStage, lastStatus, lastMessage)
}

func (r *ObservabilityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.Observability{}).
		Complete(r)
}

func (r *ObservabilityReconciler) getStages() []apiv1.ObservabilityStageName {
	return []apiv1.ObservabilityStageName{
		apiv1.GrafanaInstallation,
		apiv1.GrafanaConfiguration,
		apiv1.PrometheusInstallation,
		apiv1.PrometheusConfiguration,
		apiv1.PrometheusRules,
	}
}

func (r *ObservabilityReconciler) updateStatus(cr *apiv1.Observability, stage apiv1.ObservabilityStageName, status apiv1.ObservabilityStageStatus, lastMessage string) (ctrl.Result, error) {
	currentStatus := cr.Status.DeepCopy()

	cr.Status.Stage = stage
	cr.Status.StageStatus = status
	cr.Status.LastMessage = lastMessage

	if !reflect.DeepEqual(currentStatus, &cr.Status) {
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
		return prometheus_installation.NewReconciler(r.Client, r.Log)

	case apiv1.PrometheusConfiguration:
		return prometheus_configuration.NewReconciler(r.Client, r.Log)

	// TODO rules stage causing webhook error on create
	//case apiv1.PrometheusRules:
	//	return prometheus_rules.NewReconciler(r.Client, r.Log)

	case apiv1.GrafanaInstallation:
		return grafana_installation.NewReconciler(r.Client, r.Log)

	case apiv1.GrafanaConfiguration:
		return grafana_configuration.NewReconciler(r.Client, r.Log)

	default:
		return nil
	}
}
