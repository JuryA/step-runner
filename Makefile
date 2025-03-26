ifeq ($(shell git diff --quiet --exit-code ; echo $$?), 1)
	export STEP_RUNNER_VERSION = UNKNOWN (uncommitted changes)
else
	export STEP_RUNNER_VERSION = $(shell git rev-parse --short=8 HEAD)
endif

local := $(PWD)/.local
localBin := $(local)/bin

MODULE_NAME = gitlab.com/gitlab-org/step-runner

export PATH := $(localBin):$(PATH)

PROTOC := $(localBin)/protoc
PROTOC_VERSION := 22.2

PROTOC_GEN_GO := protoc-gen-go
PROTOC_GEN_GO_VERSION := v1.29.1

PROTOC_GEN_GO_GRPC := protoc-gen-go-grpc
PROTOC_GEN_GO_GRPC_VERSION := v1.3.0

PROTOVALIDATE_VERSION := 0.5.4
PROTOVALIDATE_DIST := $(local)/protovalidate

PROTO_SRC := proto/step.proto
PROTO_GEN := $(wildcard proto/*.pb.go)

GOIMPORTS := goimports
GOIMPORTS_VERSION := v0.23.0

# override BUILD_OS_ARCH to build for multiple platforms
LOCAL_OS_ARCH := $(lastword $(shell go version))
BUILD_OS_ARCH ?= $(LOCAL_OS_ARCH)
TARGETS := $(foreach os_arch,$(BUILD_OS_ARCH),$(os_arch))
BIN_PATH := out/bin

.DEFAULT_GOAL := precommit
.PHONY: precommit
precommit: go-fmt test

.PHONY: $(TARGETS)
$(TARGETS): GOOS=$(firstword $(subst /, ,$@))
$(TARGETS): GOARCH=$(lastword $(subst /, ,$@))
$(TARGETS): BINARY=$(BIN_PATH)/step-runner-$(subst /,-,$@)
$(TARGETS):
	@TARGET=$@ $(MAKE) build-builtin-steps

	@mkdir -p $(BIN_PATH)
	CGO_ENABLED=0 GOOS="$(GOOS)" GOARCH="$(GOARCH)" go build \
		-ldflags '-X "$(MODULE_NAME)/cmd.stepRunnerVersion=$(STEP_RUNNER_VERSION)"' \
		-o "$(BINARY)"

.PHONY: go-deps
go-deps:
	go mod download

# Build generates step runner binaries for platforms listed in BUILD_OS_ARCH.
# If there is only one platform, the binary is copied to $BIN_PATH/step-runner for ease of use.
.PHONY: build
build: FIRST_PLATFORM=$(firstword $(BUILD_OS_ARCH))
build: PLATFORM_BINARY=$(BIN_PATH)/step-runner-$(subst /,-,$(FIRST_PLATFORM))
build: FIXED_LOCATION_BINARY=$(BIN_PATH)/step-runner
build: generate go-deps $(TARGETS)
    ifeq (1, $(words $(BUILD_OS_ARCH)))
		cp $(PLATFORM_BINARY) $(FIXED_LOCATION_BINARY)
    endif

$(PROTO_GEN): $(PROTO_SRC)
	$(MAKE) generate
	if $(lastword $(shell go version))

.PHONY: .generate-proto
.generate-proto: $(PROTOC) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) $(PROTOVALIDATE_DIST)
	go generate ./proto
	$(MAKE) DIRECTORY=./proto go-fmt

.PHONY: generate
generate:
	$(MAKE) .generate-proto

.PHONY: test
test: generate build-builtin-steps
	CI_STEPS_DEBUG=true go test ./...
	@git --no-pager diff --compact-summary --exit-code -- go.mod go.sum && echo 'Go modules are tidy and complete!'
	@git --no-pager diff --compact-summary --exit-code -- ./internal/plugin/proto && echo 'proto code is up-to-date!'

.PHONY: test-race
test-race:
	go test -count 1 -race ./...

$(PROTOC): OS_TYPE ?= $(shell uname -s | tr '[:upper:]' '[:lower:]' | sed 's/darwin/osx/')
$(PROTOC): ARCH ?= $(shell uname -m | sed 's/aarch64/aarch_64/' | sed 's/arm64/aarch_64/')
$(PROTOC): DOWNLOAD_URL = https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-$(OS_TYPE)-$(ARCH).zip
$(PROTOC): OUT_DIR = $(shell dirname $(PROTOC))
$(PROTOC):
	# Installing $(DOWNLOAD_URL) as $(PROTOC)
	@mkdir -p "$(localBin)"
	@curl -sL "$(DOWNLOAD_URL)" -o "$(local)/protoc.zip"
	@unzip -u "$(local)/protoc.zip" -d "$(local)/"
	@chmod +x "$(PROTOC)"
	@rm "$(local)/protoc.zip"

.PHONY: $(PROTOC_GEN_GO)
$(PROTOC_GEN_GO):
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)

.PHONY: $(PROTOC_GEN_GO_GRPC)
$(PROTOC_GEN_GO_GRPC):
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)

$(PROTOVALIDATE_DIST): DOWNLOAD_URL = https://github.com/bufbuild/protovalidate/archive/refs/tags/v$(PROTOVALIDATE_VERSION).zip
$(PROTOVALIDATE_DIST):
	# Downloading protovalidate import from $(DOWNLOAD_URL)
	@curl -sL "$(DOWNLOAD_URL)" -o "$(local)/protovalidate.zip"
	@unzip -q -u "$(local)/protovalidate.zip" -d "$(local)/"
	@rm -fr "$(local)/protovalidate"
	@mv -f "$(local)/protovalidate-$(PROTOVALIDATE_VERSION)" "$(local)/protovalidate"
	@rm "$(local)/protovalidate.zip"

.PHONY: clean
clean: clean-builtin-steps
	@rm -rf $(BIN_PATH)

.PHONY: image
image:
	BUILD_OS_ARCH=linux/amd64 $(MAKE) build
	docker build -t step-runner -f Dockerfile.legacy .

.PHONY: check-generated
check-generated: generate
	@git --no-pager diff --compact-summary --exit-code && \
		git --no-pager diff --compact-summary --cached --exit-code

.PHONY: $(GOIMPORTS)
$(GOIMPORTS):
	@go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)

.PHONY: go-fmt
go-fmt: DIRECTORY := ./pkg ./cmd main.go
go-fmt: $(GOIMPORTS)
	$(GOIMPORTS) -w -local gitlab.com/gitlab-org/step-runner $(DIRECTORY)

.PHONY: build-builtin-steps
build-builtin-steps: TARGET ?= $(LOCAL_OS_ARCH)
build-builtin-steps: STEP_DIRS = $(shell find builtin/steps -type f -name Makefile -exec dirname {} \;)
build-builtin-steps:
	@for dir in $(STEP_DIRS); do \
		TARGET="$(TARGET)" $(MAKE) -C $$dir build; \
	done

.PHONY: clean-builtin-steps
clean-builtin-steps:
	@find builtin/bin -mindepth 1 -maxdepth 1 -type d -exec rm -rf {} \;
