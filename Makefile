GO := go
GOARCH = amd64
GOOS = linux

BUILD_FLAGS = GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOFLAGS=-mod=vendor
APPLICATION_NAME := anodot-remote-write

all: clean format build

clean:
	@rm -rf $(APPLICATION_NAME)

format:
	@echo ">> formatting code"
	@$(GO) fmt ./...

build:
	@echo ">> building binaries"
	$(BUILD_FLAGS) $(GO) build -o $(APPLICATION_NAME)