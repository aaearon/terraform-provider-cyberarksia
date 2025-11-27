.PHONY: build test testacc install lint fmt clean help check-env test-crud deps generate
.PHONY: validate validate-go validate-terraform validate-docs validate-security check-all
.PHONY: pre-commit-install pre-commit-run tools-install

BINARY_NAME=terraform-provider-cyberarksia
VERSION?=dev

# Detect OS and architecture for cross-platform support
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
INSTALL_PATH=~/.terraform.d/plugins/local/aaearon/cyberarksia/$(VERSION)/$(GOOS)_$(GOARCH)

# Default target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Install:"
	@echo "  build       - Build the provider binary"
	@echo "  install     - Install provider locally for development"
	@echo "  clean       - Clean build artifacts"
	@echo ""
	@echo "Testing:"
	@echo "  test        - Run unit tests"
	@echo "  testacc     - Run acceptance tests (requires TF_ACC=1)"
	@echo "  test-crud   - Run automated CRUD validation (usage: make test-crud DESC=resource-name)"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt         - Format Go code"
	@echo "  lint        - Run golangci-lint"
	@echo "  generate    - Generate provider documentation"
	@echo ""
	@echo "Validation (mirrors CI):"
	@echo "  validate-go         - Run all Go validation (fmt, vet, lint)"
	@echo "  validate-terraform  - Validate Terraform examples formatting"
	@echo "  validate-docs       - Check documentation is up-to-date"
	@echo "  validate-security   - Run security scans (secrets, govulncheck)"
	@echo "  validate            - Run ALL validations (RECOMMENDED before commit)"
	@echo "  check-all           - Alias for 'validate'"
	@echo ""
	@echo "Pre-commit Hooks:"
	@echo "  pre-commit-install  - Install pre-commit hooks"
	@echo "  pre-commit-run      - Run pre-commit checks manually"
	@echo ""
	@echo "Development Setup:"
	@echo "  tools-install - Install required development tools"
	@echo "  check-env     - Verify required environment variables are set"
	@echo "  deps          - Download and tidy Go dependencies"

# Build the provider binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME)

# Run unit tests
test:
	@echo "Running unit tests..."
	go test -v -race -timeout=30s ./...

# Run acceptance tests (requires TF_ACC=1)
testacc:
	@echo "Running acceptance tests..."
	TF_ACC=1 go test -v -race -timeout=120m ./internal/provider

# Install provider locally for Terraform development
install: build
	@echo "Installing provider to $(INSTALL_PATH)..."
	@mkdir -p $(INSTALL_PATH)
	@cp $(BINARY_NAME) $(INSTALL_PATH)/

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	elif [ -f ~/go/bin/golangci-lint ]; then \
		~/go/bin/golangci-lint run; \
	else \
		echo "❌ golangci-lint not found."; \
		echo "Run 'make tools-install' to install it."; \
		exit 1; \
	fi

# Format Go code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...
	gofmt -s -w .

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -rf vendor/
	@go clean -cache

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Generate documentation
generate:
	@echo "Generating documentation..."
	go generate ./...

# Check required environment variables
check-env:
	@echo "Checking required environment variables..."
	@test -n "$(CYBERARK_USERNAME)" || (echo "❌ CYBERARK_USERNAME not set (see CLAUDE.md → Environment Setup)" && exit 1)
	@test -n "$(CYBERARK_PASSWORD)" || (echo "❌ CYBERARK_PASSWORD not set (see CLAUDE.md → Environment Setup)" && exit 1)
	@echo "✅ Required environment variables are set"
	@if [ -z "$(TF_ACC)" ]; then \
		echo "⚠️  TF_ACC not set (recommended: export TF_ACC=1)"; \
	fi

# Run automated CRUD testing workflow
test-crud: check-env
	@if [ -z "$(DESC)" ]; then \
		echo "Usage: make test-crud DESC=<resource-description>"; \
		echo "Example: make test-crud DESC=policy-principal-assignment"; \
		exit 1; \
	fi
	@echo "Running CRUD validation for: $(DESC)"
	./scripts/test-crud-resource.sh "$(DESC)"

# ============================================================================
# Validation Targets (Mirror CI Checks)
# ============================================================================

# Validate Go code (formatting, vetting, linting)
validate-go:
	@echo "🔍 Validating Go code..."
	@echo "  → Checking formatting..."
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "❌ The following files are not formatted:"; \
		echo "$$unformatted"; \
		echo ""; \
		echo "Run 'make fmt' to fix formatting issues."; \
		exit 1; \
	fi
	@echo "  ✅ Formatting check passed"
	@echo "  → Running go vet..."
	@go vet ./...
	@echo "  ✅ go vet passed"
	@echo "  → Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	elif [ -f ~/go/bin/golangci-lint ]; then \
		~/go/bin/golangci-lint run --timeout=5m; \
	else \
		echo "⚠️  golangci-lint not found. Run 'make tools-install' or install manually."; \
		echo "   See: https://golangci-lint.run/welcome/install/"; \
		exit 1; \
	fi
	@echo "  ✅ golangci-lint passed"
	@echo "✅ Go validation complete!"

# Validate Terraform examples
validate-terraform:
	@echo "🔍 Validating Terraform examples..."
	@if ! command -v terraform >/dev/null 2>&1; then \
		echo "❌ terraform not found. Install from https://terraform.io"; \
		exit 1; \
	fi
	@echo "  → Checking Terraform formatting..."
	@terraform fmt -check -recursive examples/ || \
		(echo "❌ Terraform files not formatted. Run 'terraform fmt -recursive examples/' to fix." && exit 1)
	@echo "  ✅ Terraform formatting passed"
	@echo "✅ Terraform validation complete!"

# Validate documentation is up-to-date
validate-docs:
	@echo "🔍 Validating documentation..."
	@if ! command -v tfplugindocs >/dev/null 2>&1 && ! [ -f ~/go/bin/tfplugindocs ]; then \
		echo "  → Installing tfplugindocs..."; \
		go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest; \
	fi
	@echo "  → Generating documentation..."
	@if command -v tfplugindocs >/dev/null 2>&1; then \
		tfplugindocs generate --provider-name cyberarksia; \
	else \
		~/go/bin/tfplugindocs generate --provider-name cyberarksia; \
	fi
	@if [ -n "$$(git status --porcelain docs/)" ]; then \
		echo "❌ Documentation is out of date!"; \
		echo ""; \
		echo "Changed files:"; \
		git status --short docs/; \
		echo ""; \
		echo "Run 'make generate' and commit the changes."; \
		exit 1; \
	fi
	@echo "  ✅ Documentation is up-to-date"
	@echo "✅ Documentation validation complete!"

# Run security scans
validate-security:
	@echo "🔍 Running security scans..."
	@echo "  → Checking for accidentally committed secrets..."
	@if [ -f .env ]; then \
		echo "❌ .env file found in repository!"; \
		exit 1; \
	fi
	@if grep -r "CYBERARK_PASSWORD=" . --include="*.tf" --include="*.go" 2>/dev/null | grep -E -v '^[[:space:]]*#' | grep -E -v '^[[:space:]]*//'; then \
		echo "❌ Hardcoded credentials found!"; \
		exit 1; \
	fi
	@echo "  ✅ No secrets detected"
	@echo "  → Scanning dependencies for vulnerabilities..."
	@if ! command -v govulncheck >/dev/null 2>&1 && ! [ -f ~/go/bin/govulncheck ]; then \
		echo "  → Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		~/go/bin/govulncheck ./...; \
	fi
	@echo "  ✅ No vulnerabilities found"
	@echo "✅ Security validation complete!"

# Run ALL validations (recommended before commit)
validate: validate-go validate-terraform validate-docs validate-security build test
	@echo ""
	@echo "🎉 ALL validations passed!"
	@echo ""
	@echo "Your code is ready to commit. CI checks should pass."
	@echo ""

# Alias for validate
check-all: validate

# ============================================================================
# Pre-commit Hook Management
# ============================================================================

# Install pre-commit hooks
pre-commit-install:
	@if ! command -v pre-commit >/dev/null 2>&1; then \
		echo "❌ pre-commit not found."; \
		echo "Install with: pip install pre-commit"; \
		exit 1; \
	fi
	@echo "Installing pre-commit hooks..."
	@pre-commit install
	@pre-commit install --hook-type commit-msg
	@echo "✅ Pre-commit hooks installed!"
	@echo ""
	@echo "Hooks will now run automatically on 'git commit'."
	@echo "To run manually: make pre-commit-run"

# Run pre-commit checks manually
pre-commit-run:
	@if ! command -v pre-commit >/dev/null 2>&1; then \
		echo "❌ pre-commit not found."; \
		echo "Install with: pip install pre-commit"; \
		exit 1; \
	fi
	@echo "Running pre-commit checks..."
	@pre-commit run --all-files

# ============================================================================
# Development Tools Installation
# ============================================================================

# Install required development tools
tools-install:
	@echo "Installing development tools..."
	@echo ""
	@echo "→ Installing Go tools..."
	@go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "  ✅ Go tools installed"
	@echo ""
	@echo "→ Installing golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "  ✅ golangci-lint already installed ($$(golangci-lint --version))"; \
	else \
		echo "  Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin; \
		echo "  ✅ golangci-lint installed"; \
	fi
	@echo ""
	@echo "→ Installing pre-commit..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		echo "  ✅ pre-commit already installed ($$(pre-commit --version))"; \
	else \
		echo "  Install pre-commit with: pip install pre-commit"; \
		echo "  Or: brew install pre-commit (macOS)"; \
		echo "  Or: apt install pre-commit (Debian/Ubuntu)"; \
	fi
	@echo ""
	@echo "✅ Development tools setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Run 'make pre-commit-install' to enable pre-commit hooks"
	@echo "  2. Run 'make validate' to verify everything works"
