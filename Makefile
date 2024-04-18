local := $(PWD)/.local
localBin := $(local)/bin

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

SCHEMA_SRC := $(shell find schema/v1 -name "*.go")
SCHEMA_GEN := $(wildcard schema/v1/*.json)

.PHONY: build
build: $(PROTO_GEN) $(SCHEMA_GEN)
	go build .

$(PROTO_GEN): $(PROTO_SRC)
	$(MAKE) generate

$(SCHEMA_GEN): $(SCHEMA_SRC)
	$(MAKE) .generate-schema

.PHONY: .generate-schema
.generate-schema:
	go run ./schema/v1/generate

.PHONY: .generate-proto
.generate-proto: $(PROTOC) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) $(PROTOVALIDATE_DIST)
	go generate ./proto

.PHONY: generate
generate:
	$(MAKE) .generate-schema
	$(MAKE) .generate-proto

.PHONY: test
test: generate
	go test ./...
	@git --no-pager diff --compact-summary --exit-code -- go.mod go.sum && echo 'Go modules are tidy and complete!'
	@git --no-pager diff --compact-summary --exit-code -- ./internal/plugin/proto && echo 'proto code is up-to-date!'

$(PROTOC): OS_TYPE ?= $(shell uname -s | tr '[:upper:]' '[:lower:]' | sed 's/darwin/osx/')
$(PROTOC): ARCH ?= $(shell uname -m | sed 's/aarch64/aarch_64/')
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
clean:
	rm  step-runner

.PHONY: image
image:
	docker build -t step-runner .
