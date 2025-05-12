include Makefile.*.mk

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

GOLANGCI_LINT := golangci-lint
GOLANGCI_LINT_VERSION := v1.64.7

GOTESTSUM := gotestsum
GOTESTSUM_VERSION := v1.12.1

# override BUILD_OS_ARCH to build for multiple platforms
LOCAL_OS_ARCH := $(lastword $(shell go version))
BUILD_OS_ARCH ?= $(LOCAL_OS_ARCH)
PLATFORMS := $(foreach os_arch,$(BUILD_OS_ARCH),$(os_arch))
BIN_PATH := out/bin

.DEFAULT_GOAL := precommit
.PHONY: precommit
precommit: go-fmt test

.PHONY: $(PLATFORMS)
$(PLATFORMS): GOOS=$(firstword $(subst /, ,$@))
$(PLATFORMS): GOARCH=$(lastword $(subst /, ,$@))
$(PLATFORMS): BINARY=$(BIN_PATH)/step-runner-$(subst /,-,$@)
$(PLATFORMS):
	@PLATFORM=$@ $(MAKE) dist-steps-build
	@echo "Running build for step-runner"
	@mkdir -p $(BIN_PATH)
	@CGO_ENABLED=0 GOOS="$(GOOS)" GOARCH="$(GOARCH)" go build \
		-ldflags '-X "$(MODULE_NAME)/cmd.stepRunnerVersion=$(STEP_RUNNER_VERSION)"' \
		-o "$(BINARY)"

# Build generates step runner binaries for platforms listed in BUILD_OS_ARCH.
# If there is only one platform, the binary is copied to $BIN_PATH/step-runner for ease of use.
.PHONY: build
build: FIRST_PLATFORM=$(firstword $(BUILD_OS_ARCH))
build: PLATFORM_BINARY=$(BIN_PATH)/step-runner-$(subst /,-,$(FIRST_PLATFORM))
build: FIXED_LOCATION_BINARY=$(BIN_PATH)/step-runner
build: generate go-deps $(PLATFORMS)
    ifeq (1, $(words $(BUILD_OS_ARCH)))
		@cp $(PLATFORM_BINARY) $(FIXED_LOCATION_BINARY)
    endif

$(PROTO_GEN): $(PROTO_SRC)
	$(MAKE) generate
	if $(lastword $(shell go version))

.PHONY: .generate-proto
.generate-proto: $(PROTOC) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) $(PROTOVALIDATE_DIST)
	@echo "Running generate proto"
	@go generate ./proto
	@$(MAKE) DIRECTORY=./proto go-fmt

.PHONY: generate
generate:
	@$(MAKE) .generate-proto

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
clean: dist-steps-clean
	@rm -rf $(BIN_PATH)
	@find . -name report.xml | xargs rm

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

.PHONY: $(GOTESTSUM)
$(GOTESTSUM):
	@go install gotest.tools/gotestsum@$(GOTESTSUM_VERSION)

# go-deps downloads Go dependencies if the go.sum file is not empty (avoids an error message)
.PHONY: go-deps
go-deps:
	@$(MAKE) \
	DESCRIPTION="$@" \
	COMMAND='if [ -s go.sum ]; then go mod download && go mod tidy; fi' \
	run-for-all-go-modules

# go-fmt formats Go files known to git (avoids running on .local/downloaded Go files)
# The Go module name is considered a local import
.PHONY: go-fmt
go-fmt: $(GOIMPORTS)
	@$(MAKE) \
	DESCRIPTION="$@" \
	COMMAND='git ls-files "**/*.go" | xargs $(GOIMPORTS) -w -local `awk '\''NR==1{print $$$$2; exit}'\'' go.mod`' \
	run-for-all-go-modules

.PHONY: $(GOLANGCI_LINT)
$(GOLANGCI_LINT):
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: go-lint
go-lint: $(GOLANGCI_LINT)
	@$(MAKE) \
	DESCRIPTION="go-lint" \
	COMMAND='$(GOLANGCI_LINT) run --timeout 5m ./...' \
	run-for-all-go-modules

.PHONY: test
test: $(GOTESTSUM) generate dist-steps-build
	@$(MAKE) \
	DESCRIPTION="$@" \
	COMMAND='$(GOTESTSUM) --junitfile=report.xml --format=testname --rerun-fails=2 --packages="./..." -- -race ./...' \
	run-for-all-go-modules

.PHONY: dist-steps-build
dist-steps-build: PLATFORM ?= $(LOCAL_OS_ARCH)
dist-steps-build: go-deps
	@$(MAKE) dist-steps-run-make-target PLATFORM="$(PLATFORM)" MAKE_TARGET=build

.PHONY: dist-steps-clean
dist-steps-clean:
	@find dist/bin -mindepth 1 -maxdepth 1 -type d -exec rm -rf {} \;

# runs a make target in every dist step make file, if the target is present
.PHONY: dist-steps-run-make-target
dist-steps-run-make-target:
	@for dir in $(shell find dist/steps -type f -name Makefile -exec dirname {} \;); do \
		echo "Running $(MAKE_TARGET) for $$dir"; \
		$(MAKE) -q -C $$dir $(MAKE_TARGET) 2>/dev/null; \
		if [ $$? -ne 2 ]; then \
			PLATFORM="$(PLATFORM)" $(MAKE) -C $$dir $(MAKE_TARGET); \
			if [ $$? -ne 0 ]; then \
				echo "ERROR: $(MAKE_TARGET) failed."; \
				echo "       command: 'PLATFORM="$(PLATFORM)" make -C $$dir $(MAKE_TARGET)'"; \
				exit 1; \
			fi \
		fi \
	done

# runs a command for every directory that contains a go.mod
.PHONY: run-for-all-go-modules
run-for-all-go-modules: GO_MODS := $(wildcard ./go.mod */*/go.mod */*/*/go.mod */*/*/*/go.mod .gitlab/*/*/go.mod)
run-for-all-go-modules: GO_DIRS := $(dir $(GO_MODS))
run-for-all-go-modules:
	@for dir in $(GO_DIRS); do \
		echo "Running $(DESCRIPTION) for $$dir"; \
		(cd "$$dir" && $(COMMAND)) || { \
        	echo "ERROR: command failed."; \
        	echo "       command: '$(COMMAND)'"; \
        	exit 1; \
        } \
	done
