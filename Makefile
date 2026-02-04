BINARY_NAME=blast-radius-cli
VERSION=$(shell git describe --tags --always --dirty)
BUILD_FLAGS=-ldflags "-X main.Version=$(VERSION)"

all: build

build:
	go build $(BUILD_FLAGS) -o bin/$(BINARY_NAME) ./cmd/blast-radius

test:
	go test -v ./...

test-integration:
	@echo "Running integration tests..."
	go test -v ./tests/integration/...

lint:
	golangci-lint run

clean:
	rm -rf bin/
	rm -f coverage.out

fmt:
	go fmt ./...

.PHONY: all build test test-integration lint clean fmt
