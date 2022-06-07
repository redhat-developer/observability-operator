# Current Operator version
VERSION ?= 3.0.9
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Default image registry
REG ?= rhoas

# Image URL to use all building/pushing image targets
IMG ?= quay.io/$(REG)/observability-operator:v$(VERSION)

# Default bundle image tag
export BUNDLE_IMG ?= quay.io/$(REG)/observability-operator-bundle:v$(VERSION)

# Default index image tag
export INDEX_IMG ?= quay.io/$(REG)/observability-operator-index:v$(VERSION)

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,crdVersions=v1"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
ENVTEST_ASSETS_DIR = $(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

pkgs = $(shell go list ./...)

# Run unit tests
# Individual packages can be specified for testing by using the PKG argument
# For example make test/unit PKG=api/v1 
.PHONY: test/unit
test/unit: generate fmt vet manifests
	@if [ $(PKG) ]; then go test -coverprofile cover.out.tmp -tags unit ./$(PKG); else go test -coverprofile cover.out.tmp -tags unit $(pkgs); fi;
	grep -v -e "zz_generated" \
	-e "priorityclass_resources.go" \
	-e "github.com/redhat-developer/observability-operator/v3/controllers/reconcilers/configuration/alertmanager.go" \
	-e "github.com/redhat-developer/observability-operator/v3/controllers/reconcilers/configuration/configuration_reconciler.go" \
	-e "github.com/redhat-developer/observability-operator/v3/controllers/reconcilers/configuration/grafana.go" \
	-e "github.com/redhat-developer/observability-operator/v3/controllers/reconcilers/configuration/promtail.go" \
	-e "github.com/redhat-developer/observability-operator/v3/controllers/reconcilers/token/token_reconciler.go" \
	cover.out.tmp > cover.out
	rm cover.out.tmp

# Check coverage of unit tests and display by HTML 
.PHONY: test/coverage/html
test/coverage/html:
	@if [ -f cover.out ]; then go tool cover -html=cover.out; else echo "cover.out file not found"; fi;

#Check coverage of unit tests and display by standard output
.PHONY: test/coverage/output
test/coverage/output:
	@if [ -f cover.out ]; then go tool cover -func=cover.out; else echo "cover.out file not found"; fi;

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the docker image
.PHONY: docker-build
docker-build:
	docker build . -t ${IMG}

# Login to the registry
.PHONY: docker-login
docker-login:
	echo "$(QUAY_TOKEN)" | docker --config="${DOCKER_CONFIG}" login -u "${QUAY_USER}" quay.io --password-stdin

# Push the docker image
.PHONY: docker-push
docker-push:
	docker push ${IMG}

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push:
	docker push $(BUNDLE_IMG)

.PHONY: index-build
index-build:
	docker build -t $(INDEX_IMG) -f opm.Dockerfile .

.PHONY: opm-build
opm-build:
	@bash build_index.sh

.PHONY: index-push
index-push:
	docker push $(INDEX_IMG)

# deploy required secrets to cluster
NAMESPACE ?= "$(shell oc project | sed -e 's/Using project \"\([^"]*\)\".*/\1/g')"
OBSERVATORIUM_TENANT ?= "managedKafka"
OBSERVATORIUM_GATEWAY ?= "https://observatorium-mst.api.stage.openshift.com"
OBSERVATORIUM_AUTH_TYPE ?= "redhat"
OBSERVATORIUM_RHSSO_URL ?= "https://sso.redhat.com/auth/"
OBSERVATORIUM_RHSSO_REALM ?= "redhat-external"
.PHONY: deploy/secrets
deploy/secrets:
	@oc process -f ./templates/secrets-template.yml \
		-p OBSERVATORIUM_TENANT="${OBSERVATORIUM_TENANT}" \
		-p OBSERVATORIUM_GATEWAY="${OBSERVATORIUM_GATEWAY}" \
		-p OBSERVATORIUM_AUTH_TYPE="${OBSERVATORIUM_AUTH_TYPE}" \
		-p OBSERVATORIUM_RHSSO_URL="${OBSERVATORIUM_RHSSO_URL}" \
		-p OBSERVATORIUM_RHSSO_REALM="${OBSERVATORIUM_RHSSO_REALM}" \
		-p OBSERVATORIUM_RHSSO_METRICS_CLIENT_ID="${OBSERVATORIUM_RHSSO_METRICS_CLIENT_ID}" \
		-p OBSERVATORIUM_RHSSO_METRICS_SECRET="${OBSERVATORIUM_RHSSO_METRICS_SECRET}" \
		-p OBSERVATORIUM_RHSSO_LOGS_CLIENT_ID="${OBSERVATORIUM_RHSSO_LOGS_CLIENT_ID}" \
		-p OBSERVATORIUM_RHSSO_LOGS_SECRET="${OBSERVATORIUM_RHSSO_LOGS_SECRET}" \
		-p GITHUB_ACCESS_TOKEN="${GITHUB_ACCESS_TOKEN}" \
		| oc apply -f - -n $(NAMESPACE)

# deploy grafana-datasource secret required for CRC cluster
.PHONY: deploy/crc/secret
deploy/crc/secret:
	@oc process -f ./templates/crc-secret-template.yml | oc apply -f - -n openshift-monitoring
