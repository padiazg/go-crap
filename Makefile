.PHONY: build test lint

build: pkg=github.com/padiazg/go-crap/pkg/version
build: ldflags = -X $(pkg).version=$(shell git describe --tags --always --dirty) 
build: ldflags += -X $(pkg).commit=$(shell git rev-parse HEAD)
build: ldflags += -X $(pkg).buildDate=$(shell date -Iseconds)

build:
	@echo "Building go-crap..."
	@echo "ldflags: $(ldflags)"
	@go build -o go-crap -ldflags "$(ldflags)"

test:
	go test ./... -v -count=1

lint:
	golangci-lint run ./...

