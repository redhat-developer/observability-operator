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

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/jeremyary/observability-operator/api/v1"

	"github.com/openshift/library-go/pkg/manifest"
)

const (
	observabilityName string = "managed-services-observability"
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

	// gatekeep Observability instance by name so that we're only dealing with the single CR we create/intend
	if req.Name != observabilityName {
		err := fmt.Errorf("observability resource name must be '%s'", observabilityName)
		log.Error(err, "invalid Observability resource name")
		// Return success to avoid requeue
		return ctrl.Result{}, nil
	}

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
	// TODO: if/when Observability has content, we'll want to reconcile it here

	// load up & create all our various resources
	if err = r.deployResources(ctx, log, obs); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// deployResources fetches desired resources from YAML files under config/observability & attempts to create/update each
func (r *ObservabilityReconciler) deployResources(ctx context.Context, log logr.Logger, o *apiv1.Observability) error {
	var expectedResources []string
	_ = filepath.Walk("config/observability", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			expectedResources = append(expectedResources, path)
		}
		return nil
	})

	manifests, err := manifest.ManifestsFromFiles(expectedResources)
	if err != nil {
		return errors.Wrapf(err, "error while trying to fetch manifests from files")
	}
	log.Info("found manifests", "count", len(manifests))
	for _, m := range manifests {
		if err := r.updateOrCreateResource(ctx, log, &m, o); err != nil {
			return errors.Wrapf(err, "error while creating or updating resource")
		}
	}
	return nil
}

// updateOrCreateResource will take each resource loaded via YAML file from deployResources, make any value substitutions
// where needed, and create/update via client
func (r *ObservabilityReconciler) updateOrCreateResource(ctx context.Context, log logr.Logger, manifest *manifest.Manifest, o *apiv1.Observability) error {

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(manifest.Obj.GetAPIVersion())
	obj.SetKind(manifest.Obj.GetKind())

	meta := types.NamespacedName{
		Name:      manifest.Obj.GetName(),
		Namespace: manifest.Obj.GetNamespace(),
	}

	if manifest.Obj.GetNamespace() != "" {
		err := ctrl.SetControllerReference(o, manifest.Obj, r.Scheme)
		if err != nil {
			return errors.Wrapf(err, "Unable to set controller reference for %s", meta)
		}
	}

	err := r.Get(ctx, meta, obj)
	switch {
	case err == nil:
		manifest.Obj.SetResourceVersion(obj.GetResourceVersion())
		err = r.Update(ctx, manifest.Obj)
		if err != nil {
			return errors.Wrapf(err, "error updating resource %s", meta)
		}
		log.Info("resource updated", "object", meta)

	case apierrors.IsNotFound(err):
		// TODO: we can probably make this cleaner by abstraction, but decided it's good enough for initial dev
		if manifest.Obj.GetKind() == "PodMonitor" {
			namespaces := []string{"openshift-monitoring"}
			err = unstructured.SetNestedStringSlice(manifest.Obj.Object, namespaces, "spec", "namespaceSelector", "matchNames")
			if err != nil {
				return errors.Wrapf(err, "error while setting match namespaces on PodMonitor")
			}
		}
		err = r.Create(ctx, manifest.Obj)
		if err != nil {
			return errors.Wrapf(err, "error creating resource %s", meta)
		}
		log.Info("resource created", "object", meta)

	case err != nil:
		return errors.Wrapf(err, "error fetching resource %s", meta)
	}
	return err
}

func (r *ObservabilityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.Observability{}).
		Complete(r)
}
