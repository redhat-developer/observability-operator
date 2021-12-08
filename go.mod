module github.com/redhat-developer/observability-operator/v3

go 1.16

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/integr8ly/grafana-operator/v3 v3.10.3
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/operator-framework/api v0.3.20
	github.com/operator-framework/operator-registry v1.12.6-0.20200611222234-275301b779f8
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator v0.43.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.43.0
	github.com/sirupsen/logrus v1.8.1
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.2
)

replace k8s.io/client-go => k8s.io/client-go v0.19.2

replace github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.3

replace github.com/containerd/containerd => github.com/containerd/containerd v1.4.12

replace go.mongodb.org/mongo-driver => go.mongodb.org/mongo-driver v1.5.1

replace github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.2
