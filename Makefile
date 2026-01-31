# hack Makefile

# Note: We use bin/ subdirectory for build output to avoid conflict with hack/ scripts dir
BINARY_NAME := hack
BINARY := bin/$(BINARY_NAME)
PREFIX ?= /usr/local
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X github.com/cloudygreybeard/hack/cmd.Version=$(VERSION) \
	-X github.com/cloudygreybeard/hack/cmd.Commit=$(COMMIT) \
	-X github.com/cloudygreybeard/hack/cmd.Date=$(DATE)

.PHONY: all build test lint clean install snapshot help

## all: Build the binary (default target)
all: build

## build: Build the binary
build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## test: Run tests
test:
	go test -v -race ./...

## lint: Run linter
lint:
	golangci-lint run

## clean: Remove build artifacts
clean:
	rm -rf bin/
	rm -rf dist/

## install: Install to PREFIX/bin
install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 $(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY_NAME)

## snapshot: Build a snapshot release (no publish)
snapshot:
	goreleaser release --snapshot --clean

## deps: Download dependencies
deps:
	go mod download
	go mod tidy

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'
