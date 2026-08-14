# SKELETON adapted for abstrax-deploy
BINARY      := abstrax-deploy
MODULE      := github.com/useabstrax/abstrax/plugins/deploy
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE  ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w \
	-X $(MODULE)/internal/plugin.Version=$(VERSION) \
	-X $(MODULE)/internal/plugin.Commit=$(COMMIT) \
	-X $(MODULE)/internal/plugin.BuildDate=$(BUILD_DATE)

.PHONY: build build-linux-amd64 build-linux-arm64 test vet fmt-check clean manifest

build:
	go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/abstrax-deploy

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 ./cmd/abstrax-deploy

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64 ./cmd/abstrax-deploy

test:
	go test -race ./...

vet:
	go vet ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt required on:"; gofmt -l .; exit 1)

clean:
	rm -rf bin dist plugin-manifest.json

manifest: build-linux-amd64 build-linux-arm64
	@./scripts/generate-manifest.sh
