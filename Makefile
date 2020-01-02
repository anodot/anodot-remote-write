GO := go
HELM:=helm
GOFLAGS=-mod=vendor

GOARCH := amd64
GOOS := linux

GOLINT_VERSION:=1.19.1

BUILD_FLAGS = GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOFLAGS=$(GOFLAGS)
APPLICATION_NAME := anodot-prometheus-remote-write
DOCKER_IMAGE_NAME := anodot/prometheus-remote-write

VERSION := $(shell grep 'VERSION' pkg/version/version.go | awk '{ print $$4 }' | tr -d '"')
PREVIOUS_VERSION := $(shell git show HEAD:pkg/version/version.go | grep 'VERSION' | awk '{ print $$4 }' | tr -d '"' )
GIT_COMMIT := $(shell git describe --dirty --always)

all: clean format vet build-charts test build build-container test-container
publish-container: clean format vet build-charts test build build-container test-container push-container
lint: check-formatting errorcheck vet build-charts
test-all: test build build-container test-container

clean:
	@rm -rf $(APPLICATION_NAME)
	docker rmi -f `docker images $(DOCKER_IMAGE_NAME):$(VERSION) -a -q` || true
	rm -rf anodot-prometheus-remote-write-$(VERSION).tgz

check-formatting:
	./utils/check_if_formatted.sh

format:
	@$(GO) fmt ./...

vet:
	@curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b $$(go env GOPATH)/bin v$(GOLINT_VERSION)
	$(BUILD_FLAGS) $$(go env GOPATH)/bin/golangci-lint run

errorcheck: install-errcheck
	$$(go env GOPATH)/bin/errcheck ./pkg/...

install-errcheck:
	which errcheck || GO111MODULE=off go get -u github.com/kisielk/errcheck

build:
	@echo ">> building binaries with version $(VERSION)"
	$(BUILD_FLAGS) $(GO) build -ldflags "-s -w -X github.com/anodot/anodot-remote-write/pkg/version.REVISION=$(GIT_COMMIT)" -o $(APPLICATION_NAME)

build-container: build
	docker build -t $(DOCKER_IMAGE_NAME):$(VERSION) .
	@echo ">> created docker image $(DOCKER_IMAGE_NAME):$(VERSION)"

build-charts:
	$(HELM) init --client-only
	$(HELM) version --client
	@$(HELM) plugin install https://github.com/instrumenta/helm-kubeval || echo "Skipping error..."
	./utils/kubeval.sh
	$(HELM) lint deployment/helm/*
	$(HELM) package deployment/helm/*

test:
	GOFLAGS=$(GOFLAGS) $(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic -timeout 10s ./pkg/...

test-container: build-container
	@docker rm -f $(APPLICATION_NAME) || true
	@docker run -d -P --name=$(APPLICATION_NAME) $(DOCKER_IMAGE_NAME):$(VERSION) --token abc --url http://localhost
	docker ps
	set -x curl --connect-timeout 5 --max-time 10 --retry 5 --retry-delay 0 --retry-max-time 40 -I http://localhost:$$(docker port $(APPLICATION_NAME) | grep -o '[0-9]*$$' )/health

	docker logs $(APPLICATION_NAME)
	@docker rm -f $(APPLICATION_NAME)

e2e: build-container
	GOFLAGS=$(GOFLAGS) $(GO) test -v -count=1 -timeout 60s ./e2e/...

push-container: test-container
	docker push $(DOCKER_IMAGE_NAME):$(VERSION)

dockerhub-login:
	docker login -u $(DOCKER_USERNAME) -p $(DOCKER_PASSWORD)

version-set:
	@sed -i '' 's/tag: "$(PREVIOUS_VERSION)"/tag: "$(VERSION)"/g' deployment/helm/anodot-prometheus-remote-write/values.yaml && \
	sed -i '' 's/appVersion: "$(PREVIOUS_VERSION)"/appVersion: "$(VERSION)"/g' deployment/helm/anodot-prometheus-remote-write/Chart.yaml && \
	sed -i '' 's#$(DOCKER_IMAGE_NAME):$(PREVIOUS_VERSION)#$(DOCKER_IMAGE_NAME):$(VERSION)#g' deployment/docker-compose/docker-compose.yaml && \
	sed -i '' 's#$(DOCKER_IMAGE_NAME):$(PREVIOUS_VERSION)#$(DOCKER_IMAGE_NAME):$(VERSION)#g' e2e/docker-compose.yaml && \
	echo "Version $(VERSION) set in code, deployment, chart"

vendor-update:
	GO111MODULE=on go mod tidy
	GO111MODULE=on go mod vendor
