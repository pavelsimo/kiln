BINARY    := kiln
MODULE    := github.com/pavelsimo/kiln
BUILD_DIR := bin
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -ldflags "-X $(MODULE)/cmd.Version=$(VERSION)"

GOLANGCI_LINT_VERSION := v2.12.2
GOFUMPT_VERSION       := v0.7.0
GOIMPORTS_VERSION     := v0.29.0

.PHONY: build install test coverage lint fmt fmt-check docs banner ci release clean tools help

build: ## Build binary to bin/
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .

install: ## Install binary to $GOPATH/bin
	go install $(LDFLAGS) ./...

test: ## Run tests with coverage
	go test ./... -coverprofile=coverage.out -covermode=atomic

coverage: test ## Open coverage report in browser
	go tool cover -html=coverage.out

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format code with gofumpt + goimports
	gofumpt -l -w .
	goimports -l -w .

fmt-check: ## Check formatting (CI-safe, exits non-zero if dirty)
	@out=$$(gofumpt -l .); if [ -n "$$out" ]; then echo "gofumpt: $$out"; exit 1; fi
	@out=$$(goimports -l .); if [ -n "$$out" ]; then echo "goimports: $$out"; exit 1; fi

docs: ## Preview the docs site at http://localhost:8000
	python3 -m http.server 8000 -d docs

banner: ## Re-render docs/assets/banner.png from banner.html
	cd docs/assets && chromium --headless --disable-gpu --screenshot=banner.png --window-size=1280,420 --force-device-scale-factor=2 --hide-scrollbars --virtual-time-budget=8000 banner.html

ci: fmt-check lint test build ## Full CI gate

release: ## Cut a release with goreleaser (requires GITHUB_TOKEN)
	goreleaser release --clean

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR) coverage.out dist/

tools: ## Install development tools
	go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
	go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install github.com/evilmartians/lefthook@latest
	go install github.com/goreleaser/goreleaser@latest

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
