VERSION?=$(shell grep 'VERSION' cmd/flux/main.go | awk '{ print $$4 }' | tr -d '"')
EMBEDDED_MANIFESTS_TARGET=cmd/flux/manifests

rwildcard=$(foreach d,$(wildcard $(addsuffix *,$(1))),$(call rwildcard,$(d)/,$(2)) $(filter $(subst *,%,$(2)),$(d)))

all: test build

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

test: $(EMBEDDED_MANIFESTS_TARGET) tidy fmt vet
	go test ./... -coverprofile cover.out

e2e: $(EMBEDDED_MANIFESTS_TARGET) tidy fmt vet build
	go test -v --tags=e2e ./test/e2e -coverprofile cover.out

$(EMBEDDED_MANIFESTS_TARGET): $(call rwildcard,manifests/,*.yaml *.json)
	./manifests/scripts/bundle.sh

build: $(EMBEDDED_MANIFESTS_TARGET)
	CGO_ENABLED=0 go build -o ./bin/flux ./cmd/flux

install:
	go install cmd/flux

install-dev:
	CGO_ENABLED=0 go build -o /usr/local/bin ./cmd/flux
