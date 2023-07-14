SHELL = /usr/bin/env bash
COLOR := "\e[1;36m%s\e[0m\n"

SRCS = $(shell git ls-files '*.go' | grep -v '^vendor/')
VERSION=$(shell cat "./VERSION" 2> /dev/null)

TAG := $(shell git tag -l --contains HEAD)
SHA := $(shell git rev-parse --short HEAD)
GIT := $(if $(TAG),$(TAG),$(SHA))
VERSION := $(if $(VERSION),$(VERSION),$(GIT))

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
KACHE_VERSION := github.com/kacheio/kache/pkg/utils/version

GO_FLAGS := -ldflags "\
		-X $(KACHE_VERSION).Version=$(VERSION) \
		-X $(KACHE_VERSION).Branch=$(BRANCH) \
		-X $(KACHE_VERSION).Build=$(SHA)"

TEST_TIMEOUT := 20m 

.PHONY: all help verify check lint format test test-with-race mod mod-check mod-update mod-vendor clean license license-check release snap-release run build build-run

help: ## Print help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

verify: lint license-check mod-check ## Verify source (code, licencse, modules). 

check: verify test-with-race build ## Check the build (lint, test, build).

lint: ## Run linters.
	@printf $(COLOR) "Run linters..."
	@golangci-lint run --verbose --timeout 10m --fix=false --new-from-rev=HEAD~ --config=.golangci.yml

format: ## Run gofmt.
	@printf $(COLOR) "Run formatters..."
	@gofmt -s -l -w $(SRCS)

test: ## Run all unit tests.
	@go test -timeout=$(TEST_TIMEOUT) ./...

test-with-race: ## Run all unit tests with data race detect.
	@go test -timeout=$(TEST_TIMEOUT) -tags $(TEST_TAG) -race -count 1 ./...

mod: # Run go mod.
	go mod tidy

mod-check: ## Check modules.
	go mod tidy
	@git diff --exit-code go.mod

mod-update: ## Update modules.
	go get -u ./...
	go mod tidy

mod-vendor: ## Vendor modules.
	go mod vendor

clean: ## Clean test results.
	go clean -testcache

license: ## Add license header.
	go run tools/license/main.go -license=LICENSE

license-check: ## Check license header.
	go run tools/license/main.go -license=LICENSE -check

release: ## Make release.
	goreleaser release --skip-publish
	tar cfz dist/kache-${VERSION}.src.tar.gz \
		--exclude-vcs \
		--exclude dist \
		--exclude .github .
		
snap-release: ## Make snapshot release.
	goreleaser release --snapshot --clean --skip-publish

run: build ## Run dev.
	@./kache

build: ## Build.
	@go build -o kache $(GO_FLAGS) cmd/kache/main.go

build-run: build ## Build and run.
	@./kache