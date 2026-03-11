BINARY = mcp-bridge

.PHONY: build run clean vet

build:
	go build -o $(BINARY) .

run: build
	./$(BINARY)

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
