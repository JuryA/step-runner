
GOIMPORTS := goimports
GOIMPORTS_VERSION := v0.23.0

.PHONY: $(GOIMPORTS)
$(GOIMPORTS):
	@go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)

.PHONY: go-fmt
go-fmt: DIRECTORY := ./pkg main.go
go-fmt: $(GOIMPORTS)
	$(GOIMPORTS) -w -local gitlab.com/components/action-runner $(DIRECTORY)
