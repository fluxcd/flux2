VERSION?=$(shell grep 'VERSION' cmd/flux/main.go | awk '{ print $$4 }' | head -n 1 | tr -d '"')
EMBEDDED_MANIFESTS_TARGET=cmd/flux/.manifests.done
TEST_KUBECONFIG?=/tmp/flux-e2e-test-kubeconfig
ENVTEST_BIN_VERSION?=latest
KUBEBUILDER_ASSETS?=$(shell $(SETUP_ENVTEST) use -i $(ENVTEST_BIN_VERSION) -p path)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

rwildcard=$(foreach d,$(wildcard $(addsuffix *,$(1))),$(call rwildcard,$(d)/,$(2)) $(filter $(subst *,%,$(2)),$(d)))

all: test build

tidy:
	go mod tidy

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

test: $(EMBEDDED_MANIFESTS_TARGET) tidy fmt vet install-envtest
	KUBEBUILDER_ASSETS="$(KUBEBUILDER_ASSETS)" go test ./... -coverprofile cover.out --tags=unit

e2e: $(EMBEDDED_MANIFESTS_TARGET) tidy fmt vet
	TEST_KUBECONFIG=$(TEST_KUBECONFIG) go test ./cmd/flux/... -coverprofile e2e.cover.out --tags=e2e -v -failfast

test-with-kind: install-envtest
	make setup-kind
	make e2e
	make cleanup-kind

$(EMBEDDED_MANIFESTS_TARGET): $(call rwildcard,manifests/,*.yaml *.json)
	./manifests/scripts/bundle.sh
	touch $@

build: $(EMBEDDED_MANIFESTS_TARGET)
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.VERSION=$(VERSION)" -o ./bin/flux ./cmd/flux

.PHONY: install
install:
	CGO_ENABLED=0 go install ./cmd/flux

install-dev:
	CGO_ENABLED=0 go build -o /usr/local/bin ./cmd/flux

install-envtest:  setup-envtest
	 $(SETUP_ENVTEST) use $(ENVTEST_BIN_VERSION)

setup-bootstrap-patch:
	go run ./tests/bootstrap/main.go

# Find or download setup-envtest
setup-envtest:
ifeq (, $(shell which setup-envtest))
	@{ \
	set -e ;\
	SETUP_ENVTEST_TMP_DIR=$$(mktemp -d) ;\
	cd $$SETUP_ENVTEST_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-runtime/tools/setup-envtest@latest ;\
	rm -rf $$SETUP_ENVTEST_TMP_DIR ;\
	}
SETUP_ENVTEST=$(GOBIN)/setup-envtest
else
SETUP_ENVTEST=$(shell which setup-envtest)
endif
