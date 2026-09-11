VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/nimishgj/vitrine/internal/cli.Version=$(VERSION)

.PHONY: build test integration lint

build:
	go build -ldflags '$(LDFLAGS)' -o vitrine ./cmd/vitrine

test:
	go test ./...

integration:
	go test -tags integration ./... -v

lint:
	go vet ./...
