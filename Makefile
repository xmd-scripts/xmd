.PHONY: test cover build e2e

COVER_PKGS = ./internal/...

test:
	go test ./...

cover:
	go test -coverprofile=coverage.out -coverpkg=$(shell echo $(COVER_PKGS) | tr ' ' ',') $(COVER_PKGS)
	go tool cover -func=coverage.out

build:
	mkdir -p bin
	go build -ldflags="-X main.version=$$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o bin/xmd ./cmd/xmd/

e2e: build
	./test/e2e.sh
