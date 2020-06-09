VERSION?=$(shell grep 'VERSION' cmd/tk/main.go | awk '{ print $$4 }' | tr -d '"')
PROJECT_ROOT=$(shell git rev-parse --show-toplevel)

all: test build

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

test: tidy fmt vet
	go test ./... -coverprofile cover.out

build:
	CGO_ENABLED=0 go build -o $(PROJECT_ROOT)/bin/tk .

install:
	go install .

.PHONY: docs
docs:
	mkdir -p $(PROJECT_ROOT)/docs/cmd && go run . docgen $(PROJECT_ROOT)/docs/cmd

install-dev:
	CGO_ENABLED=0 go build -o /usr/local/bin .
