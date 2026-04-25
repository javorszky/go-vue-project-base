.PHONY: tidy build

tidy:
	go mod tidy
	go mod download
	go mod vendor

build: tidy
	go build -trimpath -ldflags="-s -w" -o bin/server ./cmd/server
