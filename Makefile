SHELL := /bin/sh
BIN := /usr/local/bin

REV := $(shell git rev-parse HEAD)
CHANGES := $(shell test -n "$$(git status --porcelain)" && echo '+CHANGES' || true)

PROTO_LOCATION := $(shell find proto -iname "proto" -exec echo "-I="{} \;)

PACKAGE = oda
TARGET = oda

VERSION=$(shell git describe --tags --abbrev=0 || echo "x.x.x")
COMMIT=$(shell git rev-parse --short HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

# figure out current platform
UNAME := $(shell uname | tr A-Z a-z )

# docker registry prefix:
DOCKER_REGISTRY := registry.gitlab.codilas.com/codilas/devzero/oda

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -s -w -X config.Version=${VERSION} -X config.Commit=${COMMIT}${CHANGES} -X config.Branch=${BRANCH}

CGO_CFLAGS=-I/usr/local/include
CGO_LDFLAGS=-L/usr/local/lib

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

BUF_VERSION := 1.30.0
BUF_BINARY_NAME := buf

# count processors
NPROCS:=1
OS:=$(shell uname -s)
ifeq ($(OS),Linux)
	NPROCS:=$(shell grep -c ^processor /proc/cpuinfo)
endif
ifeq ($(OS),Darwin) # Assume Mac OS X
	NPROCS:=$(shell sysctl -n hw.ncpu)
endif

.DEFAULT_GOAL := all

.PHONY: \
	all \
	build \
	deps \
	doc \
	buf \
	init \
	run \
	clean \
	lint \
	proto \
	proto-lint \
	hooks \
	hooks-install \
	docker-create-buildx \
	docker-push-buildx

all: build debug

## Build binary
build: proto
	CGO_ENABLED=0 GOOS=$(UNAME) GOARCH=$(ARCH) go build -a -tags netgo -ldflags="$(LDFLAGS)" -o "$(TARGET)" .

## Install binary to GOPATH
install: proto
	CGO_ENABLED=1 GOOS=$(UNAME) go install -a -tags netgo -ldflags="$(LDFLAGS)"

## Install binary to /usr/local/bin
install-global: proto install
	@sudo cp ${GOPATH}/bin/$(TARGET) /usr/local/bin/$(TARGET)


# Check if buf is installed and version is correct, enabled smarter handling of buf rule.
BUF_INSTALLED_VERSION := $(shell if command -v $(BIN)/$(BUF_BINARY_NAME) >/dev/null; then $(BIN)/$(BUF_BINARY_NAME) --version; fi)
VERSION_MATCH := $(findstring ${BUF_INSTALLED_VERSION},$(BUF_VERSION))
## Install buf binary
buf:
	@if [ -x "$(BIN)/$(BUF_BINARY_NAME)" ] && [ -n "$(VERSION_MATCH)" ]; then \
		echo "The required version of $(BUF_BINARY_NAME) is already installed: $(BUF_VERSION)"; \
	else \
		echo "Installing or updating $(BUF_BINARY_NAME)..."; \
		curl -sSL "https://github.com/bufbuild/buf/releases/download/v${BUF_VERSION}/${BUF_BINARY_NAME}-$(shell uname -s)-$(shell uname -m)" -o "${BIN}/${BUF_BINARY_NAME}"; \
		chmod +x $(BIN)/$(BUF_BINARY_NAME); \
	fi


## Build and debug
debug: build
	./$(TARGET)

## Install tools and deps
deps: tools
	go get -t -v ./...

## Run local godoc server on :8080
doc:
	godoc -http=localhost:8080 -index

## Install precommit hooks
hooks-install:
	pre-commit install

## Run precommit hooks
hooks:
	pre-commit run -a

## Install tools, deps, build, check
init: deps buf build

## Lint code
lint:
	go fmt ./...

## Build and run
run: build
	./$(TARGET)

## Run go tests
test:
	go test ./... -v

## Clean up all generated files
clean:
	rm -rf ./docs/*
	rm -rf ./$(TARGET)

## Lint proto files
proto-lint:
	buf lint --error-format=json

## Compile all proto files with buf
proto: clean buf
	rm -rf ./gen/*
	mkdir -p gen # create the empty directory for proto targets.
	buf generate --verbose .

## Rebuild dockerfile and run docker compose
docker-compose:
	docker-compose build
	docker-compose up

## Rebuild dockerfile and run docker compose - ubuntu
docker-compose-ubuntu:
	docker-compose -f ubuntu.docker-compose.yml build
	docker-compose -f ubuntu.docker-compose.yml up

# ## Push docker image to registry
docker-push:
	$(eval NAME=${DOCKER_REGISTRY}/${TARGET})
	$(eval DOCKER_FQDN=${NAME}:${BRANCH}-${COMMIT})
	docker build -f Dockerfile . -t ${DOCKER_FQDN}
	docker push ${DOCKER_FQDN}
	@echo "Pushed  ${DOCKER_FQDN} !"

## Create buildx remote builder for m1 support
docker-create-buildx:
	export DOCKER_HOST=tcp://10.10.150.66:2375
	docker buildx create --driver docker-container --platform linux/amd64,linux/arm64 --name remote-builder
	docker buildx use remote-builder

## Push docker image to registry with buildx
docker-push-buildx:
	$(eval NAME=${DOCKER_REGISTRY}/${TARGET})
	$(eval DOCKER_FQDN=${NAME}:${BRANCH}-${COMMIT})
	docker buildx build --platform linux/amd64,linux/arm64 -t ${DOCKER_FQDN} . --push
	@echo "Pushed  ${DOCKER_FQDN} !"

# Fancy help message
# Source: https://gist.github.com/prwhite/8168133
# COLOR
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
RESET  := $(shell tput -Txterm sgr0)
TARGET_MAX_CHAR_NUM=20

## Show help
help:
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "  ${YELLOW}%-$(TARGET_MAX_CHAR_NUM)s${RESET} ${GREEN}%s${RESET}\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)
