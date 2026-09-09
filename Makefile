BINARY := bin/collate
INSTALL_DIR ?= $(HOME)/.local/bin

DERIVED_VERSION := $(shell \
	tag=$$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null); \
	if printf '%s\n' "$$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		base=$${tag#v}; \
		if git describe --tags --exact-match --match "$$tag" >/dev/null 2>&1 && [ -z "$$(git status --porcelain)" ]; then \
			printf '%s' "$$base"; \
		else \
			major=$${base%%.*}; rest=$${base#*.}; minor=$${rest%%.*}; patch=$${rest#*.}; \
			count=$$(git rev-list --count "$$tag"..HEAD); sha=$$(git rev-parse --short HEAD); \
			dirty=; [ -z "$$(git status --porcelain)" ] || dirty=.dirty; \
			printf '%s.%s.%s-dev.%s+g%s%s' "$$major" "$$minor" "$$((patch + 1))" "$$count" "$$sha" "$$dirty"; \
		fi; \
	else \
		printf 'dev'; \
	fi)
VERSION ?= $(DERIVED_VERSION)

.PHONY: help version package install tag test validate-version validate-tag-version

help: ## List available commands
	@printf "Available commands:\n"
	@printf "  make package [VERSION=<semver>]  Build $(BINARY) with the inferred or explicit version\n"
	@printf "  make install [VERSION=<semver>]  Build and install collate into $(INSTALL_DIR)\n"
	@printf "  make test                        Run all tests\n"
	@printf "  make tag VERSION=<semver>        Create an annotated v<VERSION> Git tag\n"
	@printf "  make version                     Print the version inferred from Git\n"
	@printf "  make help                        List available commands\n"

validate-version:
	@if [ "$(VERSION)" != "dev" ] && ! printf '%s\n' "$(VERSION)" | grep -Eq '^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$$'; then \
		printf 'error: VERSION must be a semantic version such as 1.2.3 or v1.2.3\n' >&2; \
		exit 1; \
	fi

validate-tag-version:
	@if [ "$(origin VERSION)" != "command line" ]; then \
		printf 'error: VERSION is required, for example: make tag VERSION=1.2.3\n' >&2; \
		exit 1; \
	fi
	@if ! printf '%s\n' "$(VERSION)" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$$'; then \
		printf 'error: VERSION must be an unprefixed semantic version such as 1.2.3\n' >&2; \
		exit 1; \
	fi

version: validate-version ## Print the inferred or explicit version
	@printf '%s\n' "$(VERSION)"

package: validate-version ## Build the collate binary
	@mkdir -p $(dir $(BINARY))
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/collate

install: package ## Install collate for the current user
	@mkdir -p "$(INSTALL_DIR)"
	install -m 0755 $(BINARY) "$(INSTALL_DIR)/collate"
	@printf 'Installed collate to %s/collate\n' "$(INSTALL_DIR)"

tag: validate-tag-version ## Create an annotated release tag
	git tag -a "v$(VERSION)" -m "v$(VERSION)"

test: ## Run all tests
	go test ./...
