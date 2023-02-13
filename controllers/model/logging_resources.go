package model

import (
	v14 "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetLoggingSubscription(cr *v1.Observability) *v1alpha1.Subscription {
	return &v1alpha1.Subscription{
		ObjectMeta: v12.ObjectMeta{
			Name:      "cluster-logging",
			Namespace: "openshift-logging",
			Labels:    map[string]string{"app.kubernetes.io/managed-by": "observability-operator"},
		},
	}
}

func GetClusterLoggingCR() *v14.ClusterLogging {

	memoryRequests := &v13.ResourceRequirements{
		Limits:   map[v13.ResourceName]resource.Quantity{},
		Requests: map[v13.ResourceName]resource.Quantity{v13.ResourceMemory: resource.MustParse("736Mi")},
	}

	profileToleration := v13.Toleration{
		Key:      "bf2.org/kafkaInstanceProfileType",
		Operator: "Exists",
		Effect:   "NoExecute",
	}

	return &v14.ClusterLogging{
		TypeMeta: v12.TypeMeta{},
		ObjectMeta: v12.ObjectMeta{
			Name:      "instance",
			Namespace: "openshift-logging",
			Labels:    map[string]string{"app.kubernetes.io/managed-by": "observability-operator"},
		},
		Spec: v14.ClusterLoggingSpec{
			ManagementState: "Managed",
			Collection: &v14.CollectionSpec{
				Logs: &v14.LogCollectionSpec{
					Type: "fluentd",
					CollectorSpec: v14.CollectorSpec{
						Resources:   memoryRequests,
						Tolerations: []v13.Toleration{profileToleration},
					},
				},
			},
		},
	}

}

func GetClusterLogForwarderCR() *v14.ClusterLogForwarder {
	return &v14.ClusterLogForwarder{
		ObjectMeta: v12.ObjectMeta{
			Name:      "instance",
			Namespace: "openshift-logging",
		},
	}
}
