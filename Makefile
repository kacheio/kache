SHELL = /usr/bin/env bash

SRCS = $(shell git ls-files '*.go' | grep -v '^vendor/')

.PHONY: all lint format test test-with-race mod mod-check mod-vendor clean run

COLOR := "\e[1;36m%s\e[0m\n"
TEST_TIMEOUT := 20m 

lint: ## Run linters.
	@printf $(COLOR) "Run linters..."
	@golangci-lint run --verbose --timeout 10m --fix=false --new-from-rev=HEAD~ --config=.golangci.yml

format: ## Run gofmt.
	@printf $(COLOR) "Run formatters..."
	@gofmt -s -l -w $(SRCS)

test: ## Run all tests.
	@go test -timeout=$(TEST_TIMEOUT) ./...

test-with-race: ## Run all unit tests with data race detect.
	@go test -timeout=$(TEST_TIMEOUT) -tags $(TEST_TAG) -race -count 1 ./...

mod: # Run go mod.
	go mod tidy

mod-check: ## Check modules.
	go mod tidy
	@git diff --exit-code go.mod

mod-vendor: ## Vendor modules.
	go mod vendor

clean: ## Clean test results.
	go clean -testcache

run: ## Run dev.
	go run cmd/kache/main.go -config.file kache.yml