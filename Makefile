# ============================================================================
# preflight — Makefile
# ============================================================================

BINARY_NAME := preflight
BUILD_DIR   := ./bin

VERSION    ?= dev
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

.PHONY: all build test lint fmt clean tidy install run setup

all: build

## build: Compile the binary into ./bin/
build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/preflight

## test: Run all tests with race detector
test:
	go test -race -count=1 ./...

## lint: Run golangci-lint — zero-errors policy, no auto-fix
lint:
	golangci-lint run ./...

## fmt: Format source files in place via golangci-lint formatters (gofmt + goimports)
fmt:
	golangci-lint fmt ./...

## tidy: Tidy go.mod and go.sum
tidy:
	go mod tidy

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## install: Install binary to $GOPATH/bin
install:
	go install $(LDFLAGS) ./...

## run: Build and run (pass ARGS= for arguments)
run: build
	$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

## setup: One-time post-clone setup — installs Node deps and git hooks
setup:
	npm install
	lefthook install

## release-dry-run: Run goreleaser in snapshot mode (no publish)
release-dry-run:
	goreleaser release --snapshot --clean

.PHONY: release-dry-run
