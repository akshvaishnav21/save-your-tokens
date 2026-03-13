BINARY=syt
GOFLAGS=-ldflags="-s -w"
BUILD_DIR=dist

.PHONY: build test lint install clean test-race test-single

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY) .

test:
	go test ./...

lint:
	golangci-lint run ./...

install:
	CGO_ENABLED=0 go install $(GOFLAGS) .

clean:
	rm -rf $(BUILD_DIR)

test-race:
	go test -race -count=1 ./...

test-single:
	go test ./... -run $(TEST)
