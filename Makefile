.PHONY: build install test clean run help docs demo

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

DEMO_DIR=/tmp/spec-tdd-demo
DEMO_SPEC=$(DEMO_DIR)/auth-spec.md

## demo: Run kire → spec-tdd end-to-end workflow test
demo: build
	@rm -rf $(DEMO_DIR)
	@mkdir -p $(DEMO_DIR)
	@printf '%s\n' \
		'# 認証システム仕様' \
		'' \
		'## ログイン機能' \
		'' \
		'### REQ-001: ログイン認証' \
		'' \
		'ユーザーはメールアドレスとパスワードでログインできる。' \
		'' \
		'- Given: 有効なアカウントが存在する' \
		'- When: 正しいメールアドレスとパスワードでログインする' \
		'- Then: JWTトークンが返却される' \
		'' \
		'- Given: 有効なアカウントが存在する' \
		'- When: 間違ったパスワードでログインする' \
		'- Then: 401エラーが返却される' \
		'' \
		'## アカウントロック' \
		'' \
		'### REQ-002: アカウントロック機能' \
		'' \
		'連続でログインに失敗するとアカウントをロックする。' \
		'' \
		'- Given: 有効なアカウントが存在する' \
		'- When: 5回連続でログインに失敗する' \
		'- Then: アカウントがロックされる' \
		'' \
		'- Given: アカウントがロックされている' \
		'- When: 正しいパスワードでログインを試みる' \
		'- Then: ロック中のエラーメッセージが表示される' \
		'' \
		'ロック解除までの時間は30分で良いか？' \
		> $(DEMO_SPEC)
	@echo "=== [1/6] kire: Markdownを分割 ==="
	@cd $(DEMO_DIR) && kire --in auth-spec.md -o .kire --jsonl --force --quiet
	@echo "=== [2/6] spec-tdd init ==="
	@cd $(DEMO_DIR) && $(CURDIR)/$(BINARY_NAME) init
	@echo "=== [3/6] spec-tdd import kire ==="
	@cd $(DEMO_DIR) && $(CURDIR)/$(BINARY_NAME) import kire \
		--jsonl .kire/auth-spec/metadata.jsonl \
		--dir .kire/auth-spec \
		$(if $(GEMINI_API_KEY),--enrich,)
	@echo "=== [4/6] spec-tdd scaffold ==="
	@cd $(DEMO_DIR) && $(CURDIR)/$(BINARY_NAME) scaffold
	@echo "=== [5/6] spec-tdd trace ==="
	@cd $(DEMO_DIR) && $(CURDIR)/$(BINARY_NAME) trace
	@echo "=== [6/6] spec-tdd map ==="
	@cd $(DEMO_DIR) && $(CURDIR)/$(BINARY_NAME) map
	@echo ""
	@echo "=== Results ==="
	@cat $(DEMO_DIR)/.tdd/trace.md
	@echo ""
	@echo "Demo workspace: $(DEMO_DIR)"

## docs: Generate documentation
docs: build
	@echo "Generating documentation..."
	./$(BINARY_NAME) docs --format markdown --output ./docs
