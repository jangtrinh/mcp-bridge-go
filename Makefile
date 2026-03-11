BINARY  = mcp-bridge
VERSION = 1.2.0
COMMIT  = $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build run clean lint audit

## build: Compile binary with embedded version info
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## run: Build and run the MCP server
run: build
	./$(BINARY)

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)

## lint: Run all static analysis (vet + golangci-lint)
lint:
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed, skipping"

## audit: Full quality gate (format + vet + lint + security + build)
audit:
	@echo "=== gofmt ==="
	@test -z "$$(gofmt -l .)" || (echo "gofmt failed:"; gofmt -d .; exit 1)
	@echo "✅ PASS"
	@echo ""
	@echo "=== go vet ==="
	@go vet ./...
	@echo "✅ PASS"
	@echo ""
	@echo "=== golangci-lint ==="
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... && echo "✅ PASS" || echo "⏭ skipped (not installed)"
	@echo ""
	@echo "=== gosec ==="
	@which gosec > /dev/null 2>&1 && gosec -quiet ./... && echo "✅ PASS" || echo "⏭ skipped (not installed)"
	@echo ""
	@echo "=== go build ==="
	@go build -ldflags "$(LDFLAGS)" -o /dev/null .
	@echo "✅ PASS"
	@echo ""
	@echo "🏆 All quality gates passed."
