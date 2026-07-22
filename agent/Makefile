.PHONY: all build clean test install

VERSION := 1.0.0
BINARY := securelens-agent
BUILD_DIR := build
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

all: build

build:
	@echo "Building $(BINARY) for current platform..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .

build-all: clean
	@echo "Building $(BINARY) v$(VERSION) for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	@echo "  → linux/amd64"
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 .
	
	@echo "  → linux/arm64"
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 .
	
	@echo "  → darwin/amd64"
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 .
	
	@echo "  → darwin/arm64"
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 .
	
	@echo "  → windows/amd64"
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe .
	
	@echo "Done! Binaries in $(BUILD_DIR)/"

clean:
	rm -rf $(BUILD_DIR)

test:
	go test -v ./...

install: build
	@echo "Installing $(BINARY) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "Done! Run '$(BINARY) --help' to get started."

deps:
	go mod download
	go mod tidy

lint:
	golangci-lint run ./...

release: build-all
	@echo "Creating release archives..."
	@cd $(BUILD_DIR) && \
		tar -czf $(BINARY)-$(VERSION)-linux-amd64.tar.gz $(BINARY)-linux-amd64 && \
		tar -czf $(BINARY)-$(VERSION)-linux-arm64.tar.gz $(BINARY)-linux-arm64 && \
		tar -czf $(BINARY)-$(VERSION)-darwin-amd64.tar.gz $(BINARY)-darwin-amd64 && \
		tar -czf $(BINARY)-$(VERSION)-darwin-arm64.tar.gz $(BINARY)-darwin-arm64 && \
		zip $(BINARY)-$(VERSION)-windows-amd64.zip $(BINARY)-windows-amd64.exe
	@echo "Release archives created in $(BUILD_DIR)/"
