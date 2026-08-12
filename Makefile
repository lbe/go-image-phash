SHELL := /bin/bash

# Default package for tests. Override with: make test PKG=./phash/...
PKG ?= ./...

# Allow cross-compilation via GOOS/GOARCH
GOOS ?= $$(go env GOOS)
GOARCH ?= $$(go env GOARCH)
BINARY_NAME := bin/phash-$(GOOS)-$(GOARCH)

# Formatter orchestrator (see treefmt.toml): gofmt/goimports for Go,
# dprint for markdown, shfmt for bash. Override with: make format TREEFMT=dprint
TREEFMT ?= treefmt

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make build         - Build the phash binary + compile-check all packages"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make format        - Format everything via treefmt (gofmt, goimports, dprint, shfmt)"
	@echo "  make format-check  - Check formatting without writing"
	@echo "  make lint          - Run golangci-lint"
	@echo "  make cover         - Generate coverage report"
	@echo "  make bench         - Run Go benchmarks (dct, gensyncpool, phash)"
	@echo "  make bench-update  - Regenerate docs/benchmarks.txt from current toolchain"
	@echo "  make test          - Run tests for $(PKG)"
	@echo "  make test-all      - Run tests across all packages (incl. optional perl cross-check suite)"
	@echo ""
	@echo "Environment variables:"
	@echo "  PKG=<path>         - Override package (default: $(PKG))"
	@echo "  ARGS=<flags>       - Extra flags passed to go test (e.g. -v -count=1)"
	@echo "  GOOS/GOARCH        - Cross-compilation target for build"

.PHONY: test
test:
	# Extra go test flags can be passed via ARGS: make test ARGS="-count=1 -v"
	time go test $(PKG) $(ARGS)

.PHONY: test-all
test-all:
	# Run tests across all packages, including the optional perl cross-check suite.
	# The perl-tagged tests skip cleanly when Perl/Image::PHash is unavailable.
	time go test -tags perl ./... $(ARGS)

.PHONY: lint
lint:
	# Run golangci-lint across all packages
	golangci-lint run --max-same-issues 0 ./...

BENCH_PKGS := ./dct/... ./internal/gensyncpool/... ./phash/...

.PHONY: cover
cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -n 1

.PHONY: bench
bench:
	# Run package benchmarks. Pass ARGS for -count, -bench, etc.
	go test -bench=. -benchmem -count=1 -run='^$$' $(BENCH_PKGS) $(ARGS)

.PHONY: bench-update
bench-update:
	# Record benchmark output for the current Go toolchain into docs/benchmarks.txt
	@{ \
		echo "goos: $$(go env GOOS)"; \
		echo "goarch: $$(go env GOARCH)"; \
		echo "go: $$(go version | awk '{print $$3}')"; \
		if [ -r /proc/cpuinfo ]; then \
			echo "cpu: $$(grep -m1 'model name' /proc/cpuinfo | sed 's/.*: //')"; \
		fi; \
		echo; \
		go test -bench=. -benchmem -count=1 -run='^$$' $(BENCH_PKGS); \
	} > docs/benchmarks.txt
	@echo "Wrote docs/benchmarks.txt"

.PHONY: build
build:
	# Compile-check every package, then build the phash CLI binary
	# (respects GOOS/GOARCH for cross-compilation)
	go build -v ./...
	go build -v -o $(BINARY_NAME) ./cmd/phash

.PHONY: clean
clean:
	# Remove build artifacts
	rm -rf bin coverage.out coverage.html
	@echo "Cleaned build artifacts"

.PHONY: format fmt
format fmt:
	# Format everything via treefmt (gofmt+goimports for Go, dprint for markdown, shfmt for bash)
	$(TREEFMT)

.PHONY: format-check fmt-check
format-check fmt-check:
	# Fail if any file needs reformatting
	$(TREEFMT) --fail-on-change
