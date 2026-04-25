.PHONY: tidy build test test-go test-fe lint lint-go lint-fe lint-shell lint-fix fix-go fix-fe fix-shell

SHELL_SCRIPTS := $(shell find . -name '*.sh' -not -path './vendor/*' -not -path './frontend/*')

tidy:
	go mod tidy
	go mod download
	go mod vendor

build: tidy
	go build -trimpath -ldflags="-s -w" -o bin/server ./cmd/server

test: test-go test-fe

test-go:
	go test ./...

test-fe:
	cd frontend && npm run test

lint: lint-go lint-fe lint-shell

lint-go:
	golangci-lint config verify
	golangci-lint run
	golangci-lint fmt --diff

lint-fe:
	cd frontend && npm run typecheck
	cd frontend && npm run lint
	cd frontend && npm run format:check

lint-shell:
	[ -z "$(SHELL_SCRIPTS)" ] || shellcheck $(SHELL_SCRIPTS)

lint-fix: fix-go fix-fe fix-shell

fix-go:
	golangci-lint config verify
	golangci-lint run --fix
	golangci-lint fmt

fix-fe:
	cd frontend && npm run lint:fix
	cd frontend && npm run format

fix-shell:
	@diff=$$(shellcheck --format=diff $(SHELL_SCRIPTS)); \
	[ -z "$$diff" ] || echo "$$diff" | git apply
