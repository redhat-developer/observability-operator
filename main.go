/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"os"

	"github.com/go-logr/logr"
	grafana "github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	configv1 "github.com/openshift/api/config/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	coreosv1 "github.com/operator-framework/api/pkg/operators/v1"
	coreosv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apiv1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	"github.com/redhat-developer/observability-operator/v3/controllers"
	"github.com/redhat-developer/observability-operator/v3/runners"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(apiv1.AddToScheme(scheme))

	utilruntime.Must(projectv1.AddToScheme(scheme))

	utilruntime.Must(routev1.AddToScheme(scheme))

	utilruntime.Must(prometheusv1.AddToScheme(scheme))

	utilruntime.Must(coreosv1alpha1.AddToScheme(scheme))

	utilruntime.Must(coreosv1.AddToScheme(scheme))

	utilruntime.Must(grafana.AddToScheme(scheme))

	utilruntime.Must(configv1.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var disableWebhooks bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&disableWebhooks, "disable-webhooks", true, "disable webhooks for running on local environment")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "04220e3f.redhat.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	observabilityReconciler := &controllers.ObservabilityReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Observability"),
		Scheme: mgr.GetScheme(),
	}

	if err = observabilityReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Observability")
		os.Exit(1)
	}

	if !disableWebhooks {
		if err = (&apiv1.Observability{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Observability")
			os.Exit(1)
		}
	}
	// +kubebuilder:scaffold:builder
	mgr.Add(runners.NewOperandInitializer(func() error {
		if err = observabilityReconciler.InitializeOperand(mgr); err != nil {
			setupLog.Error(err, "unable to create or update operand", "controller", "Observability")
		}
		return err
	}))

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		// ENABLE TO AUTO-DELETE CR ON OPERATOR SIGINT/KILL FOR LOCAL DEV
		// if err := injectStopHandler(mgr, o, setupLog); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func injectStopHandler(mgr ctrl.Manager, o *apiv1.Observability, setupLog logr.Logger) error {
	defer func() {
		setupLog.Info("SIGINT/KILL received, deleting Observability CR")
		_ = mgr.GetClient().Delete(context.Background(), o)
	}()
	err := mgr.Start(ctrl.SetupSignalHandler())
	return err
}
