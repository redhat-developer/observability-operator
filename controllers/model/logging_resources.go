package model

import (
	v14 "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetLoggingSubscription(cr *v1.Observability) *v1alpha1.Subscription {
	return &v1alpha1.Subscription{
		ObjectMeta: v12.ObjectMeta{
			Name:      "cluster-logging",
			Namespace: "openshift-logging",
		},
	}
}

func GetClusterLoggingCR() *v14.ClusterLogging {

	memoryRequests := &v13.ResourceRequirements{
		Limits:   map[v13.ResourceName]resource.Quantity{},
		Requests: map[v13.ResourceName]resource.Quantity{v13.ResourceMemory: resource.MustParse("736Mi")},
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
						Resources: memoryRequests,
					},
				},
			},
		},
	}

}

func GetClusterLogForwarderPipeline() *v14.PipelineSpec {
	return &v14.PipelineSpec{
		OutputRefs: []string{"cloudwatch"},
		InputRefs:  []string{"kafka-operator-application-logs"},
		Name:       "observability-app-logs",
	}
}

func GetClusterLogForwarderCR() *v14.ClusterLogForwarder {

	kafkaOperatorsInput := v14.InputSpec{
		Name:           "kafka-operator-application-logs",
		Application:    &v14.Application{Namespaces: []string{"redhat-kas-fleetshard-operator", "redhat-managed-kafka-operator"}},
		Infrastructure: nil,
		Audit:          nil,
	}

	kafkaOutput := v14.OutputSpec{
		Name: "cloudwatch",
		Type: "cloudwatch",
		OutputTypeSpec: v14.OutputTypeSpec{
			Cloudwatch: &v14.Cloudwatch{
				Region:  "eu-west-1",
				GroupBy: "namespaceName",
			},
		},
		TLS: &v14.OutputTLSSpec{},
		Secret: &v14.OutputSecretSpec{
			Name: "instance",
		},
	}

	kafkaForwarder := v14.ClusterLogForwarder{
		ObjectMeta: v12.ObjectMeta{
			Name:      "instance",
			Namespace: "openshift-logging",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "observability-operator",
			},
		},
		Spec: v14.ClusterLogForwarderSpec{
			Inputs:         []v14.InputSpec{kafkaOperatorsInput},
			Outputs:        []v14.OutputSpec{kafkaOutput},
			OutputDefaults: &v14.OutputDefaults{},
		},
	}

	return &kafkaForwarder
}
