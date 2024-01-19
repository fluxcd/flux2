GO_TEST_ARGS ?=
PROVIDER_ARG ?=
TEST_TIMEOUT ?= 60m
FLUX_BINARY ?= ../../bin/flux

test: sops-check
	mkdir -p build
	cp $(FLUX_BINARY) build/flux
	# These two versions of podinfo are pushed to the cloud registry and used in tests for ImageUpdateAutomation
	docker pull ghcr.io/stefanprodan/podinfo:6.0.0
	docker pull ghcr.io/stefanprodan/podinfo:6.0.1
	go test -timeout $(TEST_TIMEOUT) -v ./ $(GO_TEST_ARGS) $(PROVIDER_ARG)

test-azure:
	$(MAKE) test PROVIDER_ARG="-provider azure" GO_TEST_ARGS="--tags azure $(GO_TEST_ARGS)"

test-gcp:
	$(MAKE) test PROVIDER_ARG="-provider gcp"

destroy:
	go test -timeout $(TEST_TIMEOUT) -v ./ $(GO_TEST_ARGS) $(PROVIDER_ARG) -destroy-only

destroy-azure:
	$(MAKE) destroy PROVIDER_ARG="-provider azure"

destroy-gcp:
	$(MAKE) destroy PROVIDER_ARG="-provider gcp"

sops-check:
ifeq ($(shell which sops),)
	$(error "no sops in PATH, consider installing")
endif
