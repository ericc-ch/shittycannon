GOLANGCI_LINT_VERSION := v2.13.2
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: check fmt lint test

check: test lint
	gofmt -l .

fmt:
	gofmt -w .

lint:
	$(GOLANGCI_LINT) run

test:
	go test ./...
