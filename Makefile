GO := go
GOARCH := amd64
GOOS := linux

BUILD_FLAGS = GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOFLAGS=-mod=vendor
APPLICATION_NAME := anodot-prometheus-remote-write
DOCKER_IMAGE_NAME := anodot/prometheus-remote-write

VERSION := $(shell grep 'VERSION' pkg/version/version.go | awk '{ print $$4 }' | tr -d '"')
PREVIOUS_VERSION := $(shell git show HEAD:pkg/version/version.go | grep 'VERSION' | awk '{ print $$4 }' | tr -d '"' )
GIT_COMMIT := $(shell git describe --dirty --always)

all: clean format vet build-charts test build build-container test-container
publish-container: clean format vet build-charts test build build-container test-container push-container
run-checks: check-formatting vet build-charts test build build-container test-container

clean:
	@rm -rf $(APPLICATION_NAME)
	docker rmi -f `docker images $(DOCKER_IMAGE_NAME):$(VERSION) -a -q` || true
	rm -rf anodot-prometheus-remote-write-$(VERSION).tgz

check-formatting:
	./utils/check_if_formatted.sh

format:
	@$(GO) fmt ./...

vet:
	@echo ">> not implemented yet..."

build:
	@echo ">> building binaries with version $(VERSION)"
	$(BUILD_FLAGS) $(GO) build -ldflags "-s -w -X github.com/anodot/anodot-prometheus-remote-write/pkg/version.REVISION=$(GIT_COMMIT)" -o $(APPLICATION_NAME)

build-container:
	docker build -t $(DOCKER_IMAGE_NAME):$(VERSION) .
	@echo ">> created docker image $(DOCKER_IMAGE_NAME):$(VERSION)"

build-charts:
	helm init --client-only
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

push-container:
	docker push $(DOCKER_IMAGE_NAME):$(VERSION)

version-set:
	@sed -i '' 's/tag: "$(PREVIOUS_VERSION)"/tag: "$(VERSION)"/g' deployment/helm/anodot-prometheus-remote-write/values.yaml && \
	sed -i '' 's/appVersion: "$(PREVIOUS_VERSION)"/appVersion: "$(VERSION)"/g' deployment/helm/anodot-prometheus-remote-write/Chart.yaml && \
	sed -i '' 's/version: "$(PREVIOUS_VERSION)"/version: "$(VERSION)"/g' deployment/helm/anodot-prometheus-remote-write/Chart.yaml && \
	sed -i '' 's#$(DOCKER_IMAGE_NAME):$(PREVIOUS_VERSION)#$(DOCKER_IMAGE_NAME):$(VERSION)#g' deployment/docker-compose/docker-compose.yaml && \
	echo "Version $(VERSION) set in code, deployment, chart"