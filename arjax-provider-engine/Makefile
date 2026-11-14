BINARY_NAME=arjax-provider
BUILD_DIR=bin

.PHONY: all build test clean install

all: test build

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) main.go

test:
	go test ./... -v

bench:
	go test ./... -bench=.

clean:
	rm -rf $(BUILD_DIR)

install: build
	mkdir -p ~/.local/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)
