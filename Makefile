VERSION?=$(shell grep 'VERSION' cmd/flux/main.go | awk '{ print $$4 }' | tr -d '"')
EMBEDDED_MANIFESTS_TARGET=cmd/flux/manifests
TEST_KUBECONFIG?=/tmp/flux-e2e-test-kubeconfig

rwildcard=$(foreach d,$(wildcard $(addsuffix *,$(1))),$(call rwildcard,$(d)/,$(2)) $(filter $(subst *,%,$(2)),$(d)))

all: test build

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

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

setup-kind:
	kind create cluster --name=flux-e2e-test --kubeconfig=$(TEST_KUBECONFIG) --config=.github/kind/config.yaml
	kubectl --kubeconfig=$(TEST_KUBECONFIG) apply -f https://docs.projectcalico.org/v3.16/manifests/calico.yaml
	kubectl --kubeconfig=$(TEST_KUBECONFIG) -n kube-system set env daemonset/calico-node FELIX_IGNORELOOSERPF=true

cleanup-kind:
	kind delete cluster --name=flux-e2e-test
	rm $(TEST_KUBECONFIG)

test: $(EMBEDDED_MANIFESTS_TARGET) tidy fmt vet
	go test ./... -coverprofile cover.out

e2e: $(EMBEDDED_MANIFESTS_TARGET) tidy fmt vet
	TEST_KUBECONFIG=$(TEST_KUBECONFIG) go test ./cmd/flux/... -coverprofile cover.out --tags=e2e -parallel=1

test-with-kind: setup-envtest
	make setup-kind
	make e2e
	make cleanup-kind

$(EMBEDDED_MANIFESTS_TARGET): $(call rwildcard,manifests/,*.yaml *.json)
	./manifests/scripts/bundle.sh

build: $(EMBEDDED_MANIFESTS_TARGET)
	CGO_ENABLED=0 go build -o ./bin/flux ./cmd/flux

install:
	go install cmd/flux

install-dev:
	CGO_ENABLED=0 go build -o /usr/local/bin ./cmd/flux
