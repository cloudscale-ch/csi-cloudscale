NAME=cloudscale-csi-plugin
OS ?= linux
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

all: test

publish: build push clean

.PHONY: bump-version
bump-version:
	@[ "${NEW_VERSION}" ] || ( echo "NEW_VERSION must be set (ex. make NEW_VERSION=v1.x.x bump-version)"; exit 1 )
	@(echo ${NEW_VERSION} | grep -E "^v") || ( echo "NEW_VERSION must be a semver ('v' prefix is required)"; exit 1 )
	@echo "Bumping VERSION from $(VERSION) to $(NEW_VERSION)"
	@echo $(NEW_VERSION) > VERSION
	@cp deploy/kubernetes/releases/csi-cloudscale-${VERSION}.yaml deploy/kubernetes/releases/csi-cloudscale-${NEW_VERSION}.yaml
	@sed -i'' -e 's/${VERSION}/${NEW_VERSION}/g' deploy/kubernetes/releases/csi-cloudscale-${NEW_VERSION}.yaml
	@sed -i'' -e 's/${VERSION}/${NEW_VERSION}/g' README.md
	$(eval NEW_DATE = $(shell date +%Y.%m.%d))
	@sed -i'' -e 's/## unreleased/## ${NEW_VERSION} - ${NEW_DATE}/g' CHANGELOG.md
	@ echo '## unreleased\n' | cat - CHANGELOG.md > temp && mv temp CHANGELOG.md
	@rm README.md-e CHANGELOG.md-e deploy/kubernetes/releases/csi-cloudscale-${NEW_VERSION}.yaml-e

.PHONY: compile
compile:
	@echo "==> Building the project"
	@env CGO_ENABLED=0 GOOS=${OS} GOARCH=amd64 go build -o cmd/cloudscale-csi-plugin/${NAME} -ldflags "$(LDFLAGS)" ${PKG}

.PHONY: test
test:
	@echo "==> Testing all packages"
	@go test -v ./...

.PHONY: test-integration
test-integration:
	@echo "==> Started integration tests"
	@env go test -count 1 -v -tags integration -timeout 20m ./test/...

.PHONY: build
build: compile
	@echo "==> Building the docker image"
	@docker build -t cloudscalech/cloudscale-csi-plugin:$(VERSION) -f cmd/cloudscale-csi-plugin/Dockerfile cmd/cloudscale-csi-plugin

.PHONY: push
push:
ifeq ($(shell [[ $(BRANCH) != "master" && $(VERSION) != "dev" ]] && echo true ),true)
	@echo "ERROR: Publishing image with a SEMVER version '$(VERSION)' is only allowed from master"
else
	@echo "==> Publishing cloudscalech/cloudscale-csi-plugin:$(VERSION)"
	@docker push cloudscalech/cloudscale-csi-plugin:$(VERSION)
	@echo "==> Your image is now available at cloudscalech/cloudscale-csi-plugin:$(VERSION)"
endif

.PHONY: clean
clean:
	@echo "==> Cleaning releases"
	@GOOS=${OS} go clean -i -x ./...
