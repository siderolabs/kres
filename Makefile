# THIS FILE WAS AUTOMATICALLY GENERATED, PLEASE DO NOT EDIT.
#
# Generated on 2020-08-11T15:54:06Z by kres dbf2a57-dirty.

# common variables

SHA := $(shell git describe --match=none --always --abbrev=8 --dirty)
TAG := $(shell git describe --tag --always --dirty)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
ARTIFACTS := _out
REGISTRY ?= docker.io
USERNAME ?= autonomy
REGISTRY_AND_USERNAME ?= $(REGISTRY)/$(USERNAME)
GOFUMPT_VERSION ?= abc0db2c416aca0f60ea33c23c76665f6e7ba0b6
GO_VERSION ?= 1.14
TESTPKGS ?= ./...
KRES_IMAGE ?= autonomy/kres:latest

# docker build settings

BUILD := docker buildx build
PLATFORM ?= linux/amd64
PROGRESS ?= auto
PUSH ?= false
CI_ARGS ?= 
COMMON_ARGS = --file=Dockerfile
COMMON_ARGS += --progress=$(PROGRESS)
COMMON_ARGS += --platform=$(PLATFORM)
COMMON_ARGS += --push=$(PUSH)
COMMON_ARGS += --build-arg=ARTIFACTS=$(ARTIFACTS)
COMMON_ARGS += --build-arg=SHA=$(SHA)
COMMON_ARGS += --build-arg=TAG=$(TAG)
COMMON_ARGS += --build-arg=USERNAME=$(USERNAME)
COMMON_ARGS += --build-arg=TOOLCHAIN=$(TOOLCHAIN)
COMMON_ARGS += --build-arg=GOFUMPT_VERSION=$(GOFUMPT_VERSION)
COMMON_ARGS += --build-arg=TESTPKGS=$(TESTPKGS)
TOOLCHAIN ?= docker.io/golang:1.14-alpine

all: lint unit-tests kres image-kres

target-%:  ## Builds the specified target defined in the Dockerfile. The build result will only remain in the build cache.
	@$(BUILD) --target=$* $(COMMON_ARGS) $(TARGET_ARGS) $(CI_ARGS) .

local-%:  ## Builds the specified target defined in the Dockerfile using the local output type. The build result will be output to the specified local destination.
	@$(MAKE) target-$* TARGET_ARGS="--output=type=local,dest=$(DEST) $(TARGET_ARGS)"

lint-golangci-lint:  ## Run golangci-lint
	@$(MAKE) target-$@

lint-gofumpt:  ## Runs gofumpt
	@$(MAKE) target-$@

.PHONY: fmt
fmt:  ## Formats the source code
	@docker run --rm -it -v $(PWD):/src -w /src golang:$(GO_VERSION) \
		bash -c "export GO111MODULE=on; export GOPROXY=https://proxy.golang.org; \
		cd /tmp && go mod init tmp && go get mvdan.cc/gofumpt/gofumports@$(GOFUMPT_VERSION) && \
		cd - && gofumports -w -local github.com/talos-systems/kres ."

.PHONY: base
base:  ## Prepare base toolchain
	@$(MAKE) target-$@

.PHONY: lint
lint: lint-golangci-lint lint-gofumpt  ## run linters

.PHONY: unit-tests
unit-tests:  ## Runs unit-tests
	@$(MAKE) local-$@ DEST=$(ARTIFACTS)

.PHONY: unit-tests-race
unit-tests-race:  ## Runs unit-tests with race detector enabled
	@$(MAKE) target-$@

.PHONY: $(ARTIFACTS)/kres
$(ARTIFACTS)/kres:  ## Build kres
	@$(MAKE) local-kres DEST=$(ARTIFACTS)

.PHONY: kres
kres: $(ARTIFACTS)/kres

.PHONY: image-kres
image-kres:  ## build image kres
	@$(MAKE) target-$@ TARGET_ARGS="--tag=$(REGISTRY)/$(USERNAME)/kres:$(TAG)"

.PHONY: rekres
rekres:
	@docker pull $(KRES_IMAGE)
	@docker run --rm -v $(PWD):/src -w /src $(KRES_IMAGE)

