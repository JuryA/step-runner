STEP_BIN_PATH := $(subst builtin/steps/,builtin/bin/,$(CURDIR))
LOCAL_OS_ARCH := $(lastword $(shell go version))
TARGET ?= $(LOCAL_OS_ARCH)

.PHONY: create-bin-step-dir
create-bin-step-dir:
	@rm -rf $(STEP_BIN_PATH)
	@mkdir -p $(STEP_BIN_PATH)
	@cp step.yml $(STEP_BIN_PATH)/step.yml

# $(TARGET) compiles a go binary for the specified target platform
.PHONY: $(TARGET)
$(TARGET): GOOS=$(firstword $(subst /, ,$@))
$(TARGET): GOARCH=$(lastword $(subst /, ,$@))
$(TARGET): BINARY=$(STEP_BIN_PATH)/run
$(TARGET):
	CGO_ENABLED=0 GOOS="$(GOOS)" GOARCH="$(GOARCH)" go build -o "$(BINARY)"

# build compiles the step into the bin directory
build: create-bin-step-dir $(TARGET)
