package controllers

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers/migration"

	infrastructure "github.com/openshift/api/config/v1"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/go-logr/logr"
	"github.com/prometheus-operator/prometheus-operator/pkg/k8sutil"
	"github.com/redhat-developer/observability-operator/v4/controllers/metrics"
	"github.com/redhat-developer/observability-operator/v4/controllers/model"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers/alertmanager_installation"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers/configuration"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers/csv"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers/grafana_configuration"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers/grafana_installation"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers/logging_installation"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers/prometheus_configuration"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers/prometheus_installation"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers/promtail_installation"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers/token"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/redhat-developer/observability-operator/v4/api/v1"
)

const (
	RequeueDelaySuccess    = 10 * time.Second
	RequeueDelayError      = 5 * time.Second
	ObservabilityFinalizer = "observability-cleanup"
	NoInitConfigMapName    = "observability-operator-no-init"
)

// ObservabilityReconciler reconciles a Observability object
type ObservabilityReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	installComplete bool
}

// +kubebuilder:rbac:groups=observability.redhat.com,resources=observabilities,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=observability.redhat.com,resources=observabilities/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=corev1,resources=configmaps,verbs=get;list;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=podmonitors;alertmanagers;prometheuses;prometheuses/finalizers;alertmanagers/finalizers;servicemonitors;prometheusrules;thanosrulers;thanosrulers/finalizers,verbs=get;list;create;update;patch;delete;watch
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions;infrastructures,verbs=get;list;watch
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,resourceNames=privileged,verbs=use
// +kubebuilder:rbac:groups=integreatly.org,resources=grafanas;grafanadashboards;grafanadatasources,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:urls=/metrics,verbs=get
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
// +kubebuilder:rbac:groups=authentication.k8s.io,resources=tokenreviews,verbs=create
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;persistentvolumes;persistentvolumeclaims,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:groups=apps,resources=deployments;daemonsets;statefulsets,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:groups=operators.coreos.com,resources=catalogsources;subscriptions;operatorgroups;clusterserviceversions,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:groups="",resources=namespaces;pods;nodes;nodes/proxy,verbs=get;list;watch;delete;create
// +kubebuilder:rbac:groups="",resources=secrets;serviceaccounts;configmaps;endpoints;services;nodes/proxy;persistentvolumes;persistentvolumeclaims,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:groups=logging.openshift.io,resources=clusterloggings;clusterlogforwarders,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update

func (r *ObservabilityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

			metrics.IncreaseTotalReconciliationsMetric(stage)
			if obs.DeletionTimestamp == nil {
				status, err = reconciler.Reconcile(ctx, obs, nextStatus)
			} else {
				status, err = reconciler.Cleanup(ctx, obs)
			}

			if err != nil {
				log.Error(err, fmt.Sprintf("reconciler error in stage %v", stage))
				nextStatus.LastMessage = err.Error()
				metrics.IncreaseFailedReconciliationsMetric(stage)
			} else {
				// Reset error message when everything went well
				nextStatus.LastMessage = ""
			}

			nextStatus.StageStatus = status

			// If a stage is not complete, do not continue with the next
			if status != apiv1.ResultSuccess {
				if obs.DeletionTimestamp == nil {
					log.Info("stack install in progress", "working stage", stage)
				} else {
					log.Info("stack cleanup in progress", "working stage", stage)
				}
				finished = false
				break
			}
		}
	}

	if obs.DeletionTimestamp == nil && finished && !r.installComplete {
		r.installComplete = true
		log.Info("stack installation complete")
	}

	// Ready for deletion?
	// Only remove the finalizer when all stages were successful
	if obs.DeletionTimestamp != nil && finished {
		log.Info("cleanup stages complete, removing finalizer")
		obs.Finalizers = []string{}
		err = r.Update(ctx, obs)
		r.installComplete = false
		return ctrl.Result{}, err
	}

	return r.updateStatus(obs, nextStatus)
}

func (r *ObservabilityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.Observability{}).
		Complete(r)
}

func (r *ObservabilityReconciler) UpdateOperand(from *apiv1.Observability, to *apiv1.Observability) error {
	originalName := from.Name
	originalVersion := from.ResourceVersion
	to.DeepCopyInto(from)
	from.Name = originalName
	from.ResourceVersion = originalVersion
	err := r.Client.Update(context.Background(), from)
	if err != nil {
		return err
	}
	return nil
}

func (r *ObservabilityReconciler) InitializeOperand(mgr ctrl.Manager) error {
	// Try to retrieve the namespace from the pod filesystem first
	r.Log.Info("determining if operand instantiation required")
	var namespace string
	namespacePath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	ns, err := ioutil.ReadFile(namespacePath)
	if err != nil {
		// If that does not work (running locally?) try the env vars
		namespace = os.Getenv("WATCH_NAMESPACE")
	} else {
		namespace = string(ns)
	}

	if namespace == "" {
		err := errors.New("unable to create operand: cannot detect operator namespace")
		return err
	}

	// controller/cache will not be ready during operator 'setup', use manager client & API Reader instead
	mgrClient := mgr.GetClient()
	apiReader := mgr.GetAPIReader()

	// don't initialise the operand if the following config map is found in the operator namespace
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NoInitConfigMapName,
			Namespace: strings.TrimSpace(namespace),
		},
	}
	key := client.ObjectKeyFromObject(configMap)

	err = apiReader.Get(context.Background(), key, configMap)
	if err == nil {
		// if there is no error that means that the config map is found.
		r.Log.Info(fmt.Sprintf("found config map '%s' so wont create observability cr", NoInitConfigMapName))
		return nil
	}
	if !k8sutil.IsResourceNotFoundError(err) {
		// report error if any other error than NotFound
		return err
	}

	runningOnCloud := false
	cluster := client.ObjectKey{
		Name: "cluster",
	}
	infra := &infrastructure.Infrastructure{}
	err = r.Get(context.Background(), cluster, infra)

	instance := observabilityInstanceWithStorage(namespace)
	// Check the infrastructure. If it's Libvirt, it is running on crc, so it will run without storage.
	if err == nil && string(infra.Status.PlatformStatus.Type) != string(infrastructure.LibvirtPlatformType) && string(infra.Status.PlatformStatus.Type) != string(infrastructure.NonePlatformType) {
		runningOnCloud = true
	} else {
		r.Log.Info("Running without Storage.")
		instance = observabilityInstanceWithoutStorage(namespace)
	}

	instances := apiv1.ObservabilityList{}
	if err := apiReader.List(context.Background(), &instances); err != nil {
		r.Log.Error(err, "failed to retrieve list of Observability instances")
		return err
	}

	found := false
	for _, existing := range instances.Items {
		if existing.Labels["managed-by"] != "observability-operator" {
			r.Log.Info("removing pre-existing operand missing or mismatching managed label")
			if err := r.UpdateOperand(&existing, &instance); err != nil {
				r.Log.Error(err, "Failed to remove pre-existing operand with missing or incorrect managed label")
				return err
			}
			found = true
		} else if (instances.Items[0].Spec.Storage == nil || instances.Items[0].Spec.Retention == "") && runningOnCloud {
			r.Log.Info("Adding retention period and storage spec to the pre-existing operand")
			if err := r.UpdateOperand(&existing, &instance); err != nil {
				return err
			}
			found = true
		} else {
			found = true
		}
	}

	if found {
		r.Log.Info("Operand with target name/namespace detected, skipping auto-create")
	} else {
		r.Log.Info("Target operand not found, instantiating default operand")
		if err := mgrClient.Create(context.Background(), &instance); err != nil {
			r.Log.Error(err, "failed to create new base operand")
			return err
		}
	}
	return nil
}

func (r *ObservabilityReconciler) getInstallationStages() []apiv1.ObservabilityStageName {
	return []apiv1.ObservabilityStageName{
		apiv1.TokenRequest,
		apiv1.PrometheusInstallation,
		apiv1.PrometheusConfiguration,
		apiv1.GrafanaInstallation,
		apiv1.GrafanaConfiguration,
		apiv1.AlertmanagerInstallation,
		apiv1.PromtailInstallation,
		apiv1.LoggingInstallation,
		apiv1.Csv,
		apiv1.Migration,
		apiv1.Configuration,
	}
}

func (r *ObservabilityReconciler) getCleanupStages() []apiv1.ObservabilityStageName {
	return []apiv1.ObservabilityStageName{
		apiv1.PrometheusConfiguration,
		apiv1.GrafanaConfiguration,
		apiv1.PrometheusInstallation,
		apiv1.GrafanaInstallation,
		apiv1.AlertmanagerInstallation,
		apiv1.PromtailInstallation,
		apiv1.LoggingInstallation,
		apiv1.Configuration,
		apiv1.TokenRequest,
		apiv1.Csv,
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

func observabilityInstanceWithStorage(namespace string) apiv1.Observability {
	return apiv1.Observability{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "observability-stack",
			Namespace: strings.TrimSpace(namespace),
			Labels:    map[string]string{"managed-by": "observability-operator"},
		},
		Spec: apiv1.ObservabilitySpec{
			ResyncPeriod: "1h",
			Retention:    "45d",
			Storage: &apiv1.Storage{
				PrometheusStorageSpec: &prometheusv1.StorageSpec{
					VolumeClaimTemplate: prometheusv1.EmbeddedPersistentVolumeClaim{
						EmbeddedObjectMetadata: prometheusv1.EmbeddedObjectMetadata{
							Name: "managed-services",
						},
						Spec: v1.PersistentVolumeClaimSpec{
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceStorage: resource.MustParse(model.PrometheusDefaultStorage),
								},
							},
						},
					},
				},
			},
			SelfContained: &apiv1.SelfContained{
				DisableBlackboxExporter: &([]bool{true})[0],
			},
			ConfigurationSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"configures": "observability-operator",
				},
			},
		},
	}
}

func observabilityInstanceWithoutStorage(namespace string) apiv1.Observability {
	return apiv1.Observability{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "observability-stack",
			Namespace: strings.TrimSpace(namespace),
			Labels:    map[string]string{"managed-by": "observability-operator"},
		},
		Spec: apiv1.ObservabilitySpec{
			ResyncPeriod: "1h",
			Retention:    "45d",
			SelfContained: &apiv1.SelfContained{
				DisableBlackboxExporter: &([]bool{true})[0],
				DisableSmtp:             &([]bool{true})[0],
			},
			ConfigurationSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"configures": "observability-operator",
				},
			},
		},
	}
}

func (r *ObservabilityReconciler) getReconcilerForStage(stage apiv1.ObservabilityStageName) reconcilers.ObservabilityReconciler {
	switch stage {
	case apiv1.PrometheusInstallation:
		return prometheus_installation.NewReconciler(r.Client, r.Log, r.Scheme)

	case apiv1.PrometheusConfiguration:
		return prometheus_configuration.NewReconciler(r.Client, r.Log)

	case apiv1.GrafanaInstallation:
		return grafana_installation.NewReconciler(r.Client, r.Log)

	case apiv1.GrafanaConfiguration:
		return grafana_configuration.NewReconciler(r.Client, r.Log)

	case apiv1.Csv:
		return csv.NewReconciler(r.Client, r.Log)

	case apiv1.TokenRequest:
		return token.NewReconciler(r.Client, r.Log)

	case apiv1.PromtailInstallation:
		return promtail_installation.NewReconciler(r.Client, r.Log)

	case apiv1.AlertmanagerInstallation:
		return alertmanager_installation.NewReconciler(r.Client, r.Log)

	case apiv1.Configuration:
		return configuration.NewReconciler(r.Client, r.Log)

	case apiv1.LoggingInstallation:
		return logging_installation.NewReconciler(r.Client, r.Log)

	case apiv1.Migration:
		return migration.NewReconciler(r.Client, r.Log)

	default:
		return nil
	}
}
