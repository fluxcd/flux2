VERSION?=$(shell grep 'VERSION' cmd/flux/main.go | awk '{ print $$4 }' | head -n 1 | tr -d '"')
DEV_VERSION?=0.0.0-$(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --short HEAD)-$(shell date +%s)
EMBEDDED_MANIFESTS_TARGET=cmd/flux/.manifests.done
TEST_KUBECONFIG?=/tmp/flux-e2e-test-kubeconfig
# Architecture to use envtest with
ENVTEST_ARCH ?= amd64

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

rwildcard=$(foreach d,$(wildcard $(addsuffix *,$(1))),$(call rwildcard,$(d)/,$(2)) $(filter $(subst *,%,$(2)),$(d)))

all: test build

tidy:
	go mod tidy -compat=1.20
	cd tests/integration && go mod tidy -compat=1.20

fmt:
	go fmt ./...

vet:
	go vet ./...

setup-kind:
	kind create cluster --name=flux-e2e-test --kubeconfig=$(TEST_KUBECONFIG) --config=.github/kind/config.yaml
	kubectl --kubeconfig=$(TEST_KUBECONFIG) apply -f https://docs.projectcalico.org/v3.16/manifests/calico.yaml
	kubectl --kubeconfig=$(TEST_KUBECONFIG) -n kube-system set env daemonset/calico-node FELIX_IGNORELOOSERPF=true

cleanup-kind:
	kind delete cluster --name=flux-e2e-test
	rm $(TEST_KUBECONFIG)

KUBEBUILDER_ASSETS?="$(shell $(ENVTEST) --arch=$(ENVTEST_ARCH) use -i $(ENVTEST_KUBERNETES_VERSION) --bin-dir=$(ENVTEST_ASSETS_DIR) -p path)"
TEST_PKG_PATH="./..."
test: $(EMBEDDED_MANIFESTS_TARGET) tidy fmt vet install-envtest
	KUBEBUILDER_ASSETS="$(KUBEBUILDER_ASSETS)" go test $(TEST_PKG_PATH) -coverprofile cover.out --tags=unit $(TEST_ARGS)

E2E_TEST_PKG_PATH="./cmd/flux/..."
e2e: $(EMBEDDED_MANIFESTS_TARGET) tidy fmt vet
	TEST_KUBECONFIG=$(TEST_KUBECONFIG) go test $(E2E_TEST_PKG_PATH) -coverprofile e2e.cover.out --tags=e2e -v -failfast $(TEST_ARGS)

test-with-kind: install-envtest
	make setup-kind
	make e2e
	make cleanup-kind

$(EMBEDDED_MANIFESTS_TARGET): $(call rwildcard,manifests/,*.yaml *.json)
	./manifests/scripts/bundle.sh
	touch $@

build: $(EMBEDDED_MANIFESTS_TARGET)
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.VERSION=$(VERSION)" -o ./bin/flux ./cmd/flux

build-dev: $(EMBEDDED_MANIFESTS_TARGET)
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.VERSION=$(DEV_VERSION)" -o ./bin/flux ./cmd/flux

.PHONY: install
install:
	CGO_ENABLED=0 go install ./cmd/flux

install-dev:
	CGO_ENABLED=0 go build -o /usr/local/bin ./cmd/flux

setup-bootstrap-patch:
	go run ./tests/bootstrap/main.go

setup-image-automation:
	cd tests/image-automation && go run main.go

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
ENVTEST_KUBERNETES_VERSION?=latest
install-envtest: setup-envtest
	mkdir -p ${ENVTEST_ASSETS_DIR}
	$(ENVTEST) use $(ENVTEST_KUBERNETES_VERSION) --arch=$(ENVTEST_ARCH) --bin-dir=$(ENVTEST_ASSETS_DIR)

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
setup-envtest: ## Download envtest-setup locally if necessary.
	# replace the commit SHA with 'latest' when https://github.com/kubernetes-sigs/controller-runtime/issues/2720 is fixed
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@c7e1dc9b5302d649d5531e19168dd7ea0013736d)

# go-install-tool will 'go install' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
