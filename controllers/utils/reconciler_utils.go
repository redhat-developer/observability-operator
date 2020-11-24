package utils

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	observabilityv1 "github.com/jeremyary/observability-operator/api/v1"
	v12 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetNS gets the specified corev1.Namespace from the k8s API server
func GetNS(ctx context.Context, namespace string, client k8sclient.Client) (*v1.Namespace, error) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: ns.Name}, ns)
	if err == nil {
		// workaround for https://github.com/kubernetes/client-go/issues/541
		ns.TypeMeta = metav1.TypeMeta{Kind: "Namespace", APIVersion: metav1.SchemeGroupVersion.Version}
	}
	return ns, err
}

func CreateNSWithProjectRequest(ctx context.Context, namespace string, client k8sclient.Client) (*v1.Namespace, error) {
	projectRequest := &v12.ProjectRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	if err := client.Create(ctx, projectRequest); err != nil {
		return nil, fmt.Errorf("could not create %s ProjectRequest: %v", projectRequest.Name, err)
	}

	// when a namespace is created using the ProjectRequest object it drops labels and annotations
	// so we need to retrieve the project as namespace and add them
	ns, err := GetNS(ctx, namespace, client)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve %s namespace: %v", ns.Name, err)
	}

	if err := client.Update(ctx, ns); err != nil {
		return nil, fmt.Errorf("failed to update the %s namespace definition: %v", ns.Name, err)
	}

	return ns, err
}

func ReconcileNamespace(ctx context.Context, client k8sclient.Client, namespace string) (observabilityv1.ObservabilityStageStatus, error) {
	ns, err := GetNS(ctx, namespace, client)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsForbidden(err) {
			return observabilityv1.ResultFailed, err
		}

		ns, err := CreateNSWithProjectRequest(ctx, namespace, client)
		if err != nil {
			return observabilityv1.ResultFailed, fmt.Errorf("failed to create namespace %s", ns.Name)
		}

		return observabilityv1.ResultSuccess, nil
	}

	if ns.Status.Phase == v1.NamespaceTerminating {
		return observabilityv1.ResultInProgress, nil
	}

	if ns.Status.Phase != v1.NamespaceActive {
		return observabilityv1.ResultInProgress, nil
	}

	return observabilityv1.ResultSuccess, nil
}

func PtrToInt32(i int32) *int32 {
	return &i
}

func IsRouteReads(route *routev1.Route) bool {
	if route == nil {
		return false
	}
	// A route has a an array of Ingress where each have an array of conditions
	for _, ingress := range route.Status.Ingress {
		for _, condition := range ingress.Conditions {
			// A successful route will have the admitted condition type as true
			if condition.Type == routev1.RouteAdmitted && condition.Status != v1.ConditionTrue {
				return false
			}
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
