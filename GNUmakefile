# GNUmakefile for terraform-provider-grouper
#
# Common developer tasks. Acceptance tests (testacc) talk to a real Grouper
# deployment and are gated behind TF_ACC plus explicit environment variables;
# they never run as part of `make test`.

BINARY  := terraform-provider-grouper
VERSION := dev

# Location Terraform searches for locally-installed providers.
# Adjust OS_ARCH if you are not on darwin_arm64.
OS_ARCH        ?= $(shell go env GOOS)_$(shell go env GOARCH)
INSTALL_HOST   := registry.terraform.io
INSTALL_NS     := notlimahrelyt
INSTALL_TYPE   := grouper
INSTALL_DIR    := $(HOME)/.terraform.d/plugins/$(INSTALL_HOST)/$(INSTALL_NS)/$(INSTALL_TYPE)/0.0.0-dev/$(OS_ARCH)

.PHONY: default fmt fmt-check lint vet test testacc build install generate tidy clean

default: fmt vet test

## fmt: format all Go source
fmt:
	gofmt -s -w .

## fmt-check: fail if any Go source is not gofmt-clean
fmt-check:
	@unformatted=$$(gofmt -s -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not gofmt-clean:"; echo "$$unformatted"; exit 1; \
	fi

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint (install: https://golangci-lint.run/)
lint:
	golangci-lint run ./...

## test: run unit tests (no network / no acceptance tests)
test:
	go test ./... -count=1

## testacc: run acceptance tests against a real Grouper deployment.
## Requires TF_ACC=1 and GROUPER_BASE_URL / GROUPER_USERNAME / GROUPER_PASSWORD /
## GROUPER_TEST_STEM to be set. See docs/guides for the safety rules.
testacc:
	TF_ACC=1 go test ./... -v -count=1 -timeout 120m

## build: compile the provider binary
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) .

## install: build and install into the local Terraform plugin directory for
## manual testing with a dev_overrides block or filesystem mirror.
install: build
	mkdir -p "$(INSTALL_DIR)"
	cp "$(BINARY)" "$(INSTALL_DIR)/$(BINARY)_v0.0.0-dev"
	@echo "Installed to $(INSTALL_DIR)"

## generate: regenerate registry documentation with tfplugindocs
generate:
	go generate ./...

## tidy: tidy module dependencies
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	rm -f "$(BINARY)"
	rm -rf dist/
