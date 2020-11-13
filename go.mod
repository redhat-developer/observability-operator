module github.com/jeremyary/observability-operator

go 1.13

require (
	github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/openshift/library-go v0.0.0-20201109112824-093ad3cf6600
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.43.0
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime v0.6.2
)
