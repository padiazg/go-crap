MODULE   := $(shell head -1 go.mod | awk '{print $$2}')
BINARY   := $(notdir $(MODULE))
VERSION_PKG := $(MODULE)/pkg/version

VERSION    := $(shell git describe --tags --always --dirty)
COMMIT     := $(shell git rev-parse HEAD)
BUILD_DATE := $(shell date -Iseconds)

LDFLAGS  := -X $(VERSION_PKG).version=$(VERSION)
LDFLAGS  += -X $(VERSION_PKG).commit=$(COMMIT)
LDFLAGS  += -X $(VERSION_PKG).buildDate=$(BUILD_DATE)

# ── Docker ──
DOCKER_REGISTRIES ?= docker.io/padiazg/go-crap ghcr.io/padiazg/go-crap
DOCKER_TAG        ?= latest

.PHONY: build test lint clean fmt mod-tidy help install coverage \
        fieldalignment preflight docker-build docker-push

build:
	@echo "Building $(BINARY)..."
	@go build -o $(BINARY) -ldflags "$(LDFLAGS)"

test:
	go test -race -count=1 ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY) coverage.out mutation*.json

fmt:
	gofmt -s -w .

crap:
	go run main.go scan --exclude ".*_test.go" --fail-above --threshold 30 --verbose --top 10 

mod-tidy:
	go mod tidy

install: build
	cp $(BINARY) $(shell go env GOPATH)/bin/

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

fieldalignment:
	fieldalignment -test=false ./...	

preflight: lint test fieldalignment crap

docker-build:
	docker build \
		-t go-crap:local \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		.

docker-push:
	tags=""; \
	for reg in $(DOCKER_REGISTRIES); do \
		tags="$$tags -t $${reg}:$(DOCKER_TAG)"; \
	done; \
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		$$tags --push .

docker-run:
	docker run --rm -v "$(PWD):/code" go-crap:local

help:
	@echo "build     - compile binary"
	@echo "test      - run tests with race detector"
	@echo "lint      - run golangci-lint"
	@echo "clean     - remove build artifacts"
	@echo "fmt       - format source code"
	@echo "mod-tidy  - tidy go.mod"
	@echo "install   - build and copy to GOPATH/bin"
	@echo "coverage  - generate coverage report"
	@echo "help      - show this help"
