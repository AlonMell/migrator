.PHONY: build test clean lint help all

GO=go
GOFLAGS=-trimpath
LDFLAGS=-s -w
MODULE=github.com/AlonMell/migrator

build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" ./...

test:
	$(GO) test -v ./...

clean:
	$(GO) clean
	rm -f coverage.out

lint:
	golangci-lint run


all: clean lint test build

# Help
help:
	@echo "Available targets:"
	@echo "  build       - Build the package"
	@echo "  test        - Run integration tests with Docker"
	@echo "  clean       - Clean build artifacts"
	@echo "  lint        - Run linter"
	@echo "  all         - Run clean, lint, test, cover, and build"
	@echo "  help        - Show this help message"