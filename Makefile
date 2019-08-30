GO := go
GOARCH = amd64
GOOS = linux

BUILD_FLAGS = GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOFLAGS=-mod=vendor
APPLICATION_NAME := anodot-prometheus-remote-write
DOCKER_IMAGE_NAME:= anodot/prometheus-remote-write

VERSION:=$(shell grep 'VERSION' pkg/version/version.go | awk '{ print $$4 }' | tr -d '"')
PREVIOUS_VERSION:=$(git show HEAD:pkg/version/version.go | grep 'VERSION' | awk '{ print $$4 }' | tr -d '"' )
GIT_COMMIT:=$(shell git describe --dirty --always)

all: clean format vet build-charts test build build-container test-container

clean:
	@rm -rf $(APPLICATION_NAME)
	docker rmi -f `docker images $(DOCKER_IMAGE_NAME):$(VERSION) -a -q` || true
	rm -rf anodot-prometheus-remote-write-$(VERSION).tgz

format:
	@echo ">> formatting code"
	@$(GO) fmt ./...

vet:
	@echo ">> not implemented yet..."

build:
	@echo ">> building binaries with version $(VERSION)"
	$(BUILD_FLAGS) $(GO) build -ldflags "-s -w -X github.com/anodot/anodot-remote-write/pkg/version.REVISION=$(GIT_COMMIT)" -o $(APPLICATION_NAME)

build-container:
	docker build -t $(DOCKER_IMAGE_NAME):$(VERSION) .
	@echo ">> created docker image $(DOCKER_IMAGE_NAME):$(VERSION)"

build-charts:
	helm lint deployment/helm/*
	helm package deployment/helm/*

test:
	GO111MODULE=on go test -v -race ./...

test-container:
	@docker rm -f $(APPLICATION_NAME) || true
	@docker run -d -P --name=$(APPLICATION_NAME) $(DOCKER_IMAGE_NAME):$(VERSION)
	docker ps
	curl --connect-timeout 5 --max-time 10 --retry 5 --retry-delay 0 --retry-max-time 40 -I http://localhost:$$(docker port $(APPLICATION_NAME) | grep -o '[0-9]*$$' )/health

	@docker rm -f $(APPLICATION_NAME)

version-set:
	@next="$(VERSION)" && \
	current="$(PREVIOUS_VERSION)" && \
	sed -i '' "s/tag: $$current/tag: $$next/g" deployment/helm/anodot-remote-write/values.yaml && \
	sed -i '' "s/appVersion: $$current/appVersion: $$next/g" deployment/helm/anodot-remote-write/Chart.yaml && \
	sed -i '' "s/version: $$current/version: $$next/g" deployment/helm/anodot-remote-write/Chart.yaml && \
	echo "Version $$next set in code, deployment, chart"