.PHONY: build install test clean run help docs

# Build variables
BINARY_NAME=spec-tdd
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "-X 'github.com/thirdlf03/spec-tdd/cmd.Version=$(VERSION)' \
                  -X 'github.com/thirdlf03/spec-tdd/cmd.Commit=$(COMMIT)' \
                  -X 'github.com/thirdlf03/spec-tdd/cmd.BuildDate=$(BUILD_DATE)'"

## help: Display this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

## install: Install the binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) .

## test: Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	rm -rf dist/

## run: Build and run the application
run: build
	./$(BINARY_NAME)

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## lint: Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## docs: Generate documentation
docs: build
	@echo "Generating documentation..."
	./$(BINARY_NAME) docs --format markdown --output ./docs
