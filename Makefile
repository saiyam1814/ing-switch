.PHONY: all build build-ui build-go clean test run-ui install help

BINARY := ing-switch
VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

## all: Build everything (UI + binary)
all: build

## build: Build UI then embed into Go binary
build: build-ui build-go

## build-ui: Build the React UI (output to pkg/server/dist)
build-ui:
	@echo "Building React UI..."
	cd web && npm install --silent && npm run build
	@echo "UI built → pkg/server/dist/"

## build-go: Compile the Go binary
build-go:
	@echo "Building Go binary..."
	go build $(LDFLAGS) -o $(BINARY) .
	@echo "Binary built → ./$(BINARY)"

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf web/dist pkg/server/dist/*
	@echo "Cleaned build artifacts"

## test: Run Go tests
test:
	go test ./... -v

## run-ui: Start the local UI (requires cluster access)
run-ui: build
	./$(BINARY) ui

## run-scan: Quick scan of your current cluster
run-scan: build
	./$(BINARY) scan

## install: Install binary to /usr/local/bin
install: build
	cp $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"

## dev-ui: Run UI in dev mode (Vite dev server, requires ing-switch ui running on :8080)
dev-ui:
	cd web && npm run dev

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/^## //'

# Go mod tidy
tidy:
	go mod tidy
