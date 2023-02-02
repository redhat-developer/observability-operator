package utils

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	v13 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	v12 "github.com/operator-framework/api/pkg/operators/v1"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	"github.com/redhat-developer/observability-operator/v4/controllers/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Returns the cluster id by querying the ClusterVersion resource
func GetClusterId(ctx context.Context, client k8sclient.Client) (string, error) {
	v := &v13.ClusterVersion{}
	selector := k8sclient.ObjectKey{
		Name: "version",
	}

	err := client.Get(ctx, selector, v)
	if err != nil {
		return "", err
	}

	return string(v.Spec.ClusterID), nil
}

// returns cluster Openshift version
func GetClusterOSVersion(ctx context.Context, client k8sclient.Client) (string, error) {
	v := &v13.ClusterVersion{}
	selector := k8sclient.ObjectKey{
		Name: "version",
	}

	err := client.Get(ctx, selector, v)
	if err != nil {
		return "", err
	}
	return v.Status.Desired.Version, nil
}

// We need to figure out if a sync set needs to be created
// When installing via subscription this is not required because OLM will create one
// When installing by deployment we need to create one ourselves
func HasOperatorGroupForNamespace(ctx context.Context, client k8sclient.Client, ns string) (bool, error) {
	list := &v12.OperatorGroupList{}
	opts := &k8sclient.ListOptions{
		Namespace: ns,
	}
	err := client.List(ctx, list, opts)
	if err != nil {
		return false, err
	}

	for _, group := range list.Items {
		for _, namespace := range group.Spec.TargetNamespaces {
			if namespace == ns {
				return true, nil
			}
		}
	}

	return false, nil
}

func IsRouteReady(route *routev1.Route) bool {
	if route == nil {
		return false
	}
	// A route has a an array of Ingress where each have an array of conditions
	for _, ingress := range route.Status.Ingress {
		for _, condition := range ingress.Conditions {
			// A successful route will have the admitted condition type as true
			if condition.Type == routev1.RouteAdmitted && condition.Status != corev1.ConditionTrue {
				return false
			}
		}
	}
	return true
}

func IsServiceReady(service *corev1.Service) bool {
	if service == nil {
		return false
	}

	for _, condition := range service.Status.Conditions {
		if condition.Type == "Ready" && condition.Status != metav1.ConditionTrue {
			return false
		}
	}
	return true
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(s int) string {
	b := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b)
}

func WaitForGrafanaToBeRemoved(ctx context.Context, cr *v1.Observability, client k8sclient.Client) (v1.ObservabilityStageStatus, error) {
	list := &appsv1.DeploymentList{}
	opts := &k8sclient.ListOptions{
		Namespace: cr.Namespace,
	}
	err := client.List(ctx, list, opts)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	for _, ss := range list.Items {
		if ss.Name == "grafana-deployment" {
			return v1.ResultInProgress, nil
		}
	}

	return v1.ResultSuccess, nil
}

func WaitForAlertmanagerToBeRemoved(ctx context.Context, cr *v1.Observability, client k8sclient.Client) (v1.ObservabilityStageStatus, error) {
	list := &appsv1.StatefulSetList{}
	opts := &k8sclient.ListOptions{
		Namespace: cr.Namespace,
	}
	err := client.List(ctx, list, opts)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	alertmanager := model.GetAlertmanagerCr(cr)

	for _, ss := range list.Items {
		if ss.Name == fmt.Sprintf("alertmanager-%s", alertmanager.Name) {
			return v1.ResultInProgress, nil
		}
	}

	return v1.ResultSuccess, nil
}

func WaitForPrometheusToBeRemoved(ctx context.Context, cr *v1.Observability, client k8sclient.Client) (v1.ObservabilityStageStatus, error) {
	list := &appsv1.StatefulSetList{}
	opts := &k8sclient.ListOptions{
		Namespace: cr.Namespace,
	}
	err := client.List(ctx, list, opts)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	prom := model.GetPrometheus(cr)

	for _, ss := range list.Items {
		if ss.Name == fmt.Sprintf("prometheus-%s", prom.Name) {
			return v1.ResultInProgress, nil
		}
	}

	return v1.ResultSuccess, nil
}

func RunningLocally() bool {
	// check for cluster namespace
	namespacePath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	_, err := os.ReadFile(namespacePath)
	if err != nil {
		return os.IsNotExist(err)
	}
	return false
}
