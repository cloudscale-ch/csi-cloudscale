NAME=cloudscale-csi-plugin
OS ?= linux
GO_VERSION := $(shell awk '/^go/ {print $$2}' go.mod)
ifeq ($(strip $(shell git status --porcelain 2>/dev/null)),)
  GIT_TREE_STATE=clean
else
  GIT_TREE_STATE=dirty
endif
COMMIT ?= $(shell git rev-parse HEAD)
BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
LDFLAGS ?= -X github.com/cloudscale-ch/csi-cloudscale/driver.version=${VERSION} -X github.com/cloudscale-ch/csi-cloudscale/driver.commit=${COMMIT} -X github.com/cloudscale-ch/csi-cloudscale/driver.gitTreeState=${GIT_TREE_STATE}
PKG ?= github.com/cloudscale-ch/csi-cloudscale/cmd/cloudscale-csi-plugin

VERSION ?= $(shell cat VERSION)
CHART_VERSION ?= $(shell awk '/^version:/ {print $$2}' charts/csi-cloudscale/Chart.yaml)
DOCKER_REPO ?= quay.io/cloudscalech/cloudscale-csi-plugin

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Host OS/ARCH used to namespace tool binaries so a shared ./bin (e.g. host tree
# mounted into a Linux sandbox) never mixes incompatible binaries.
HOST_PLATFORM := $(shell go env GOOS)-$(shell go env GOARCH)

## Location to install dependencies to (per-platform to keep host & sandbox separate)
LOCALBIN := $(shell pwd)/bin/$(HOST_PLATFORM)
$(LOCALBIN):
	mkdir -p "$(LOCALBIN)"

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php
# Based on Kubebuilders help target: https://github.com/kubernetes-sigs/kubebuilder/blob/f98c0ef7f0df9f83a3044d829effc336b126aaf6/pkg/plugins/golang/v4/scaffolds/internal/templates/makefile.go#L102-L117

.PHONY: all
all: lint test ## Run all checks (lint + vet + test)

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: update-k8s
update-k8s: ## Update Kubernetes version in go.mod
	scripts/update-k8s.sh $(NEW_KUBERNETES_VERSION)
	sed -i.sedbak "s/^KUBERNETES_VERSION.*/KUBERNETES_VERSION ?= $(NEW_KUBERNETES_VERSION)/" Makefile
	rm -f Makefile.sedbak

.PHONY: bump-version
bump-version: ## Bump app version (e.g. make NEW_VERSION=v4.1.0 bump-version)
	@[ "${NEW_VERSION}" ] || ( echo "NEW_VERSION must be set (ex. make NEW_VERSION=v1.x.x bump-version)"; exit 1 )
	@(echo ${NEW_VERSION} | grep -E "^v") || ( echo "NEW_VERSION must be a semver ('v' prefix is required)"; exit 1 )
	@echo "Bumping VERSION from $(VERSION) to $(NEW_VERSION)"
	@echo $(NEW_VERSION) > VERSION
	@sed -i'' -e 's/${VERSION}/${NEW_VERSION}/g' README.md
	@sed -i'' -e 's/${VERSION}/${NEW_VERSION}/g' charts/csi-cloudscale/values.yaml
	@sed -i'' -e 's/${VERSION:v%=%}/${NEW_VERSION:v%=%}/g' charts/csi-cloudscale/Chart.yaml
	@helm template csi-cloudscale -n kube-system --set nameOverride=csi-cloudscale --set renderNamespace=true ./charts/csi-cloudscale > deploy/kubernetes/releases/csi-cloudscale-${NEW_VERSION}.yaml
	$(eval NEW_DATE = $(shell date +%Y.%m.%d))
	@sed -i'' -e 's/## unreleased/## ${NEW_VERSION} - ${NEW_DATE}/g' CHANGELOG.md
	@ echo '## unreleased\n' | cat - CHANGELOG.md > temp && mv temp CHANGELOG.md
	@rm README.md-e CHANGELOG.md-e charts/csi-cloudscale/Chart.yaml-e charts/csi-cloudscale/values.yaml-e

.PHONY: bump-chart-version
bump-chart-version: ## Bump Helm chart version (e.g. make NEW_CHART_VERSION=1.6.0 bump-chart-version)
	@[ "${NEW_CHART_VERSION}" ] || ( echo "NEW_CHART_VERSION must be set (ex. make NEW_CHART_VERSION=v1.x.x bump-version)"; exit 1 )
	@(echo ${NEW_CHART_VERSION} | grep -E "^v") || ( echo "NEW_CHART_VERSION must be a semver ('v' prefix is required)"; exit 1 )
	@echo "Bumping CHART_VERSION from $(CHART_VERSION) to $(NEW_CHART_VERSION)"
	@sed -i'' -e 's/${CHART_VERSION:v%=%}/${NEW_CHART_VERSION:v%=%}/g' charts/csi-cloudscale/Chart.yaml
	@rm charts/csi-cloudscale/Chart.yaml-e

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: vet ## Run tests.
	@echo "==> Testing all packages"
	go test -v ./... $(TESTARGS)

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "==> Started integration tests"
	@env go test -race -count 1 -v $(TESTARGS) -tags integration -parallel 4 -timeout 20m ./test/...

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	"$(GOLANGCI_LINT)" run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	"$(GOLANGCI_LINT)" run --fix

.PHONY: govulncheck
govulncheck: govulncheck-tool ## Run govulncheck (advisory only)
	"$(GOVULNCHECK)" ./...

##@ Build

.PHONY: compile
compile: ## Build the project binary
	@echo "==> Building the project"
	@docker run --rm -it -e GOOS=${OS} -e GOARCH=amd64 -v ${PWD}/:/app -w /app golang:${GO_VERSION}-alpine sh -c 'apk add git && go build -o cmd/cloudscale-csi-plugin/${NAME} -ldflags "$(LDFLAGS)" ${PKG}'

.PHONY: build
build: compile ## Build the docker image
	@echo "==> Building the docker image"
	@docker build --platform linux/amd64 -t $(DOCKER_REPO):$(VERSION) cmd/cloudscale-csi-plugin -f cmd/cloudscale-csi-plugin/Dockerfile

.PHONY: push
push: ## Push docker image to registry
ifeq ($(DOCKER_REPO),quay.io/cloudscalech/cloudscale-csi-plugin)
  ifeq ($(filter master,$(BRANCH))$(filter release/%,$(BRANCH)),)
    ifneq ($(VERSION),dev)
	  $(error "Only the `dev` tag can be published from non-master/non-release branches")
    endif
  endif
endif
	@echo "==> Publishing $(DOCKER_REPO):$(VERSION)"
	@docker push $(DOCKER_REPO):$(VERSION)
	@echo "==> Your image is now available at $(DOCKER_REPO):$(VERSION)"

.PHONY: publish
publish: build push clean ## Build, push, and clean

.PHONY: clean
clean: ## Clean build artifacts
	@echo "==> Cleaning releases"
	@GOOS=${OS} go clean -i -x ./...

.PHONY: helm-template
helm-template: ## Generate helm template
	@helm template csi-cloudscale -n kube-system --set nameOverride=csi-cloudscale ./charts/csi-cloudscale

##@ Tooling

## Tool Versions
GOLANGCI_LINT_VERSION ?= v2.13.1
GOVULNCHECK_VERSION ?= v1.7.0

## Tool Binaries
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOVULNCHECK ?= $(LOCALBIN)/govulncheck

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: govulncheck-tool
govulncheck-tool: $(GOVULNCHECK) ## Download govulncheck locally if necessary.
$(GOVULNCHECK): $(LOCALBIN)
	$(call go-install-tool,$(GOVULNCHECK),golang.org/x/vuln/cmd/govulncheck,$(GOVULNCHECK_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] && [ "$$(readlink -- "$(1)" 2>/dev/null)" = "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f "$(1)" ;\
GOBIN="$(LOCALBIN)" go install $${package} ;\
mv "$(LOCALBIN)/$$(basename "$(1)")" "$(1)-$(3)" ;\
} ;\
ln -sf "$$(realpath "$(1)-$(3)")" "$(1)"
endef
