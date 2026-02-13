VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)
BINARY  := repoguide

.PHONY: build test lint fmt clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	goimports -w .

clean:
	rm -f $(BINARY) coverage.out coverage.html

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
