.PHONY: build test lint clean install run-local

# Binary name and paths
BINARY_NAME=gitops-kustomz
BIN_DIR=bin
MAIN_PATH=./cmd/gitops-kustomz

# Build variables
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

build:
	@mkdir -p ${BIN_DIR}
	go build ${LDFLAGS} -o ${BIN_DIR}/${BINARY_NAME} ${MAIN_PATH}

install: build
	cp ${BIN_DIR}/${BINARY_NAME} $(GOPATH)/bin/

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-coverage: test
	go tool cover -html=coverage.txt -o coverage.html

lint:
	golangci-lint run ./...

clean:
	rm -rf ${BIN_DIR}
	rm -f coverage.txt coverage.html
	rm -rf dist/

run: build
	${BIN_DIR}/${BINARY_NAME}

# Run in local mode with test data
run-local: build
	${BIN_DIR}/${BINARY_NAME} --run-mode local \
		--service my-app \
		--environments stg,prod \
		--lc-before test/local/before/services/my-app/environments \
		--lc-after test/local/after/services/my-app/environments \
		--policies-path sample/policies \
		--lc-output-dir test/output
	@echo ""
	@echo "📄 Reports generated:"
	@ls -lh test/output/*.md

# OPA policy tests
test-policies:
	opa test sample/policies/*.opa

# Format code
fmt:
	go fmt ./...
	gofumpt -w .

# Check for security issues
security:
	gosec ./...

# Run all checks (lint + test + security)
check: lint test security test-policies

help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  install        - Install the binary to GOPATH/bin"
	@echo "  test           - Run tests with coverage"
	@echo "  test-coverage  - Generate HTML coverage report"
	@echo "  lint           - Run linter"
	@echo "  clean          - Clean build artifacts"
	@echo "  run-local      - Run in local mode with test data"
	@echo "  test-policies  - Test OPA policies"
	@echo "  fmt            - Format code"
	@echo "  security       - Check for security issues"
	@echo "  check          - Run all checks"


