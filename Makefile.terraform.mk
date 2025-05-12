TF_LINT_VERSION = 0.57.0
TF_LINT = $(localBin)/tflint_$(TF_LINT_VERSION)
TF_MODULES := $(shell find . -name "main.tf" -not -path "*/\.terraform/*" -exec dirname {} \; | sort -u)

$(TF_LINT): OS_TYPE ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
$(TF_LINT): ARCH ?= $(shell uname -m | sed 's/aarch64/arm64/' | sed 's/x86_64/amd64/' )
$(TF_LINT): DOWNLOAD_URL = https://github.com/terraform-linters/tflint/releases/download/v$(TF_LINT_VERSION)/tflint_$(OS_TYPE)_$(ARCH).zip
$(TF_LINT):
	@mkdir -p "$(localBin)"
	@curl -sL "$(DOWNLOAD_URL)" -o "$(local)/tflint.zip"
	@unzip "$(local)/tflint.zip" -d "$(local)"
	@mv "$(local)/tflint" "$(TF_LINT)"
	@chmod +x "$(TF_LINT)"
	@rm "$(local)/tflint.zip"

tf-init: $(TF_MODULES:%=%-tf-init)
%-tf-init: MODULE=$*
%-tf-init:
	@cd $(MODULE) && terraform init -backend=false

tf-clean: $(TF_MODULES:%=%-tf-clean)
%-tf-clean: MODULE=$*
%-tf-clean:
	@rm -rf "$(MODULE)/.terraform" "$(MODULE)/.terraform.lock.hcl"

.PHONY: tf-lint
tf-lint: $(TF_LINT) tf-init $(TF_MODULES:%=%-tf-lint) tf-clean
%-tf-lint: MODULE=$*
%-tf-lint:
	@echo "Running tf-lint for $(MODULE)"
	@$(TF_LINT) --recursive --config "$$(pwd)/.tflint.hcl" --chdir=$(MODULE)

.PHONY: tf-fmt
tf-fmt: $(TF_MODULES:%=%-tf-fmt)
%-tf-fmt: MODULE=$*
%-tf-fmt:
	@echo "Running tf-fmt for $(MODULE)"
	@terraform fmt -check -recursive -diff "$(MODULE)"
