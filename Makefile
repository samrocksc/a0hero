# A0Hero — Makefile

VERSION  ?= dev
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILDDATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS  := -s -w \
  -X github.com/samrocksc/a0hero/version.Version=$(VERSION) \
  -X github.com/samrocksc/a0hero/version.Commit=$(COMMIT) \
  -X github.com/samrocksc/a0hero/version.BuildDate=$(BUILDDATE)

BINARY   := a0hero
DIST_DIR := dist

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-14s\033[0m %s\n", $$1, $$2}'

## ========== DEVELOPMENT ==========

install: ## Install dependencies
	go mod download
	go mod tidy

tidy: ## Tidy go modules
	go mod tidy

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run linters
	golangci-lint run ./...
	@go fmt ./...

## ========== BUILDING ==========

build: ## Build for current platform
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/a0hero/

build-debug: ## Build without optimizations (for dlv/delve)
	go build -gcflags "all=-N -l" -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/a0hero/

clean: ## Remove build artifacts
	rm -rf bin/ $(DIST_DIR)/

## ========== TESTING ==========

test: ## Run all tests
	go test ./... -v

test-cover: ## Run tests with coverage
	go test ./... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

ci-test: ## Run tests as CI would (no cache)
	go test ./... -v -count=1 -race

## ========== CROSS-COMPILATION ==========

# Darwin (macOS)
dist/darwin-arm64/$(BINARY):
	@mkdir -p dist/darwin-arm64
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/darwin-arm64/$(BINARY) ./cmd/a0hero/

dist/darwin-amd64/$(BINARY):
	@mkdir -p dist/darwin-amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/darwin-amd64/$(BINARY) ./cmd/a0hero/

# Linux
dist/linux-arm64/$(BINARY):
	@mkdir -p dist/linux-arm64
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/linux-arm64/$(BINARY) ./cmd/a0hero/

dist/linux-amd64/$(BINARY):
	@mkdir -p dist/linux-amd64
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/linux-amd64/$(BINARY) ./cmd/a0hero/

# Windows
dist/windows-amd64/$(BINARY).exe:
	@mkdir -p dist/windows-amd64
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/windows-amd64/$(BINARY).exe ./cmd/a0hero/

dist-all: dist/darwin-arm64/$(BINARY) dist/darwin-amd64/$(BINARY) dist/linux-arm64/$(BINARY) dist/linux-amd64/$(BINARY) dist/windows-amd64/$(BINARY).exe ## Build all platforms

## ========== ARCHIVES (for releases) ==========

archive-darwin-arm64: dist/darwin-arm64/$(BINARY)
	@mkdir -p $(DIST_DIR)
	tar czf $(DIST_DIR)/a0hero_$(VERSION)_darwin-arm64.tar.gz -C dist/darwin-arm64 $(BINARY)

archive-darwin-amd64: dist/darwin-amd64/$(BINARY)
	@mkdir -p $(DIST_DIR)
	tar czf $(DIST_DIR)/a0hero_$(VERSION)_darwin-amd64.tar.gz -C dist/darwin-amd64 $(BINARY)

archive-linux-arm64: dist/linux-arm64/$(BINARY)
	@mkdir -p $(DIST_DIR)
	tar czf $(DIST_DIR)/a0hero_$(VERSION)_linux-arm64.tar.gz -C dist/linux-arm64 $(BINARY)

archive-linux-amd64: dist/linux-amd64/$(BINARY)
	@mkdir -p $(DIST_DIR)
	tar czf $(DIST_DIR)/a0hero_$(VERSION)_linux-amd64.tar.gz -C dist/linux-amd64 $(BINARY)

archive-windows-amd64: dist/windows-amd64/$(BINARY).exe
	@mkdir -p $(DIST_DIR)
	cd dist/windows-amd64 && zip -j ../../$(DIST_DIR)/a0hero_$(VERSION)_windows-amd64.zip $(BINARY).exe

release-archives: archive-darwin-arm64 archive-darwin-amd64 archive-linux-arm64 archive-linux-amd64 archive-windows-amd64 ## Build all release archives

## ========== RUNNING ==========

run: ## Run a0hero
	go run ./cmd/a0hero

run-debug: ## Run with debug logging
	go run ./cmd/a0hero --debug

## ========== VERSION ==========

version: ## Print version info
	@go run -ldflags "$(LDFLAGS)" ./cmd/a0hero/ --version

## ========== CI/CD (used by GitHub Actions) ==========

ci: ci-test build ## Full CI: test + build

smoke: ## Quick smoke test — build + version
	@$(MAKE) build
	@./bin/$(BINARY) --version