VERSION?=$(shell grep 'VERSION' cmd/gotk/main.go | awk '{ print $$4 }' | tr -d '"')

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
	CGO_ENABLED=0 go build -o ./bin/gotk ./cmd/gotk

install:
	go install cmd/gotk

.PHONY: docs
docs:
	mkdir -p ./docs/cmd && go run ./cmd/gotk/ docgen

install-dev:
	CGO_ENABLED=0 go build -o /usr/local/bin ./cmd/gotk
