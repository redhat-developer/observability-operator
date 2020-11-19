package utils

import (
	"context"
	"fmt"
	observabilityv1 "github.com/jeremyary/observability-operator/api/v1"
	v12 "github.com/openshift/api/project/v1"
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
