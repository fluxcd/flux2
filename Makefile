VERSION?=$(shell grep 'VERSION' cmd/tk/main.go | awk '{ print $$4 }' | tr -d '"')

all: tidy fmt vet test build

build:
	CGO_ENABLED=0 go build -o ./bin/tk ./cmd/tk

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

test:
	go test ./... -coverprofile cover.out

