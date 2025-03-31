STEP_BIN_PATH := $(subst dist/steps/,dist/bin/,$(CURDIR))
LOCAL_OS_ARCH := $(lastword $(shell go version))
TARGET ?= $(LOCAL_OS_ARCH)

.PHONY: create-bin-step-dir
create-bin-step-dir:
	@rm -rf $(STEP_BIN_PATH)
	@mkdir -p $(STEP_BIN_PATH)
	@cp run_script.sh $(STEP_BIN_PATH)/run_script.sh
	@cp step.yml $(STEP_BIN_PATH)/step.yml

build: create-bin-step-dir
