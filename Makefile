BINARY := archivyy
BIN_DIR := bin

.DEFAULT_GOAL := help

## help: list available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | awk -F': ' '{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

## run: run the server
run:
	go run .

## build: compile to bin/
build:
	go build -o $(BIN_DIR)/$(BINARY) .

## test: run tests
test:
	go test ./...

## fmt: format all Go source
fmt:
	gofmt -w .

## vet: report suspicious constructs
vet:
	go vet ./...

## tidy: sync go.mod with imports
tidy:
	go mod tidy

## check: fmt, vet and test
check: fmt vet test

## lint: golangci-lint, if installed
lint:
	@command -v golangci-lint >/dev/null || { echo "golangci-lint not installed: https://golangci-lint.run/welcome/install/"; exit 1; }
	golangci-lint run

## clean: remove build output
clean:
	rm -rf $(BIN_DIR)

.PHONY: help run build test fmt vet tidy check lint clean
