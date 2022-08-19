module github.com/redhat-developer/observability-operator/v3

go 1.16

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v1.2.3
	github.com/goccy/go-yaml v1.9.5
	github.com/integr8ly/grafana-operator/v3 v3.10.3
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.17.0
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/operator-framework/api v0.3.20
	github.com/operator-framework/operator-registry v1.12.6-0.20200611222234-275301b779f8
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator v0.43.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.55.1
	github.com/sirupsen/logrus v1.8.1
	k8s.io/api v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.12.1
	github.com/prometheus/client_golang v1.13.0
	github.com/blang/semver v3.5.1+incompatible
)

require (
	github.com/Shopify/logrus-bugsnag v0.0.0-20171204204709-577dee27f20d // indirect
	github.com/bshuster-repo/logrus-logstash-hook v1.0.2 // indirect
	github.com/bugsnag/bugsnag-go v2.1.2+incompatible // indirect
	github.com/bugsnag/panicwrap v1.3.4 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/garyburd/redigo v1.6.3 // indirect
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/openshift/cluster-logging-operator v0.0.0-20221018233744-cd9347d3efbe
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.7 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.15 // indirect
)

replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.5.13
	github.com/docker/distribution => github.com/docker/distribution v2.8.0+incompatible
	github.com/go-logr/logr => github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr => github.com/go-logr/zapr v0.1.0
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.2
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.1.2
	github.com/prometheus-operator/prometheus-operator => github.com/prometheus-operator/prometheus-operator v0.43.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring => github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.43.0
	go.mongodb.org/mongo-driver => go.mongodb.org/mongo-driver v1.5.1
	k8s.io/api => k8s.io/api v0.19.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.2
	k8s.io/apiserver => k8s.io/apiserver v0.19.2
	k8s.io/client-go => k8s.io/client-go v0.19.2
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.0.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/controller-runtime/pkg/client/fake => sigs.k8s.io/controller-runtime/pkg/client/fake v0.12.1
	github.com/operator-framework/api => github.com/operator-framework/api v0.3.20
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.2
)
