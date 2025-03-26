STEP_BIN_PATH := $(subst builtin/steps/,builtin/bin/,$(CURDIR))
LOCAL_OS_ARCH := $(lastword $(shell go version))
PLATFORM ?= $(LOCAL_OS_ARCH)

.PHONY: create-bin-step-dir
create-bin-step-dir:
	@rm -rf $(STEP_BIN_PATH)
	@mkdir -p $(STEP_BIN_PATH)
	@cp step.yml $(STEP_BIN_PATH)/step.yml

# $(PLATFORM) compiles a go binary for the specified target platform
.PHONY: $(PLATFORM)
$(PLATFORM): GOOS=$(firstword $(subst /, ,$@))
$(PLATFORM): GOARCH=$(lastword $(subst /, ,$@))
$(PLATFORM): BINARY=$(STEP_BIN_PATH)/run
$(PLATFORM):
	@CGO_ENABLED=0 GOOS="$(GOOS)" GOARCH="$(GOARCH)" go build -o "$(BINARY)"

# build compiles the step into the bin directory
.PHONY: build
build: create-bin-step-dir $(PLATFORM)
