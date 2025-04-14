.PHONY: build run lint fmt clean test

BINARY_DIR=bin
BINARY_NAME=sync_with_cloud
BINARY_PATH=$(BINARY_DIR)/$(BINARY_NAME)
MAIN=cmd/main.go

run: build
	@$(BINARY_PATH)

build:
	@go build -o $(BINARY_PATH) $(MAIN)

lint: fmt
	@golangci-lint run

fmt:
	@go fmt ./...

test:
	@go test ./...
	
test-verbose:
	@go test -v ./...

clean:
	@rm -rf $(BINARY_DIR)