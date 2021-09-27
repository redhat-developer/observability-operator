module github.com/redhat-developer/observability-operator/v3

go 1.13

require (
	github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3 v3.0.5
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.2.1
	github.com/integr8ly/grafana-operator/v3 v3.10.3
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/operator-framework/api v0.3.20
	github.com/operator-framework/operator-registry v1.12.6-0.20200611222234-275301b779f8
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator v0.43.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.43.0
	github.com/sirupsen/logrus v1.6.0
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.2
)

replace k8s.io/client-go => k8s.io/client-go v0.19.2
