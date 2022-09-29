GO_TEST_ARGS ?=
PROVIDER_ARG ?=
TEST_TIMEOUT ?= 60m

test:
	go test -timeout $(TEST_TIMEOUT) -v ./ $(GO_TEST_ARGS) $(PROVIDER_ARG) --tags=integration

test-azure:
	$(MAKE) test PROVIDER_ARG="-provider azure"
