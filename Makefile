.PHONY: tidy build test test-go test-fe lint lint-go lint-fe lint-fix fix-go fix-fe

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

lint: lint-go lint-fe

lint-go:
	golangci-lint run
	golangci-lint fmt --diff

lint-fe:
	cd frontend && npm run typecheck
	cd frontend && npm run lint
	cd frontend && npm run format:check

lint-fix: fix-go fix-fe

fix-go:
	golangci-lint run --fix
	golangci-lint fmt

fix-fe:
	cd frontend && npm run lint:fix
	cd frontend && npm run format
