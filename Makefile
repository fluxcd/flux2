VERSION?=$(shell grep 'VERSION' cmd/flux/main.go | awk '{ print $$4 }' | tr -d '"')

all: test build

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

test: tidy fmt vet docs
	go test ./... -coverprofile cover.out

build:
	CGO_ENABLED=0 go build -o ./bin/flux ./cmd/flux

install:
	go install cmd/flux

.PHONY: docs
docs:
	rm docs/cmd/*
	mkdir -p ./docs/cmd && go run ./cmd/flux/ docgen

install-dev:
	CGO_ENABLED=0 go build -o /usr/local/bin ./cmd/flux
