NAME=cloudscale-csi-plugin
OS ?= linux
GO_VERSION := 1.15.5
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
CHART_VERSION ?= $(shell awk '/version:/ {print $$2}' charts/csi-cloudscale/Chart.yaml)
DOCKER_REPO ?= quay.io/cloudscalech/cloudscale-csi-plugin

all: check-unused test

publish: build push clean

.PHONY: update-k8s
update-k8s:
	scripts/update-k8s.sh $(NEW_KUBERNETES_VERSION)
	sed -i.sedbak "s/^KUBERNETES_VERSION.*/KUBERNETES_VERSION ?= $(NEW_KUBERNETES_VERSION)/" Makefile
	rm -f Makefile.sedbak

.PHONY: bump-version
bump-version:
	@[ "${NEW_VERSION}" ] || ( echo "NEW_VERSION must be set (ex. make NEW_VERSION=v1.x.x bump-version)"; exit 1 )
	@(echo ${NEW_VERSION} | grep -E "^v") || ( echo "NEW_VERSION must be a semver ('v' prefix is required)"; exit 1 )
	@echo "Bumping VERSION from $(VERSION) to $(NEW_VERSION)"
	@echo $(NEW_VERSION) > VERSION
	@sed -i'' -e 's/${VERSION}/${NEW_VERSION}/g' README.md
	@sed -i'' -e 's/${VERSION}/${NEW_VERSION}/g' charts/csi-cloudscale/values.yaml
	@sed -i'' -e 's/${VERSION:v%=%}/${NEW_VERSION:v%=%}/g' charts/csi-cloudscale/Chart.yaml
	@helm install csi-cloudscale --dry-run -n kube-system --set legacyName=true ./charts/csi-cloudscale > deploy/kubernetes/releases/csi-cloudscale-${NEW_VERSION}.yaml
	$(eval NEW_DATE = $(shell date +%Y.%m.%d))
	@sed -i'' -e 's/## unreleased/## ${NEW_VERSION} - ${NEW_DATE}/g' CHANGELOG.md
	@ echo '## unreleased\n' | cat - CHANGELOG.md > temp && mv temp CHANGELOG.md
	@rm README.md-e CHANGELOG.md-e charts/csi-cloudscale/Chart.yaml-e charts/csi-cloudscale/values.yaml-e

.PHONY: bump-chart-version
bump-chart-version:
	@[ "${NEW_CHART_VERSION}" ] || ( echo "NEW_CHART_VERSION must be set (ex. make NEW_CHART_VERSION=v1.x.x bump-version)"; exit 1 )
	@(echo ${NEW_CHART_VERSION} | grep -E "^v") || ( echo "NEW_CHART_VERSION must be a semver ('v' prefix is required)"; exit 1 )
	@echo "Bumping CHART_VERSION from $(CHART_VERSION) to $(NEW_CHART_VERSION)"
	@sed -i'' -e 's/${CHART_VERSION:v%=%}/${NEW_CHART_VERSION:v%=%}/g' charts/csi-cloudscale/Chart.yaml
	@rm charts/csi-cloudscale/Chart.yaml-e

.PHONY: compile
compile:
	@echo "==> Building the project"
	@docker run --rm -it -e GOOS=${OS} -e GOARCH=amd64 -v ${PWD}/:/app -w /app golang:${GO_VERSION}-alpine sh -c 'apk add git && go build -mod=vendor -o cmd/cloudscale-csi-plugin/${NAME} -ldflags "$(LDFLAGS)" ${PKG}'

.PHONY: check-unused
check-unused: vendor
	@git diff --exit-code -- go.sum go.mod vendor/ || ( echo "there are uncommitted changes to the Go modules and/or vendor files -- please run 'make vendor' and commit the changes first"; exit 1 )

.PHONY: test
test:
	@echo "==> Testing all packages"
	@GO111MODULE=on go test -mod=vendor -v ./...

.PHONY: test-integration
test-integration:
	@echo "==> Started integration tests"
	@env GO111MODULE=on go test -mod=vendor -count 1 -v $(TESTARGS) -tags integration -timeout 20m ./test/...

.PHONY: build
build: compile
	@echo "==> Building the docker image"
	@docker build -t $(DOCKER_REPO):$(VERSION) cmd/cloudscale-csi-plugin -f cmd/cloudscale-csi-plugin/Dockerfile

.PHONY: push
push:
ifeq ($(DOCKER_REPO),cloudscalech/cloudscale-csi-plugin)
  ifneq ($(BRANCH),master)
    ifneq ($(VERSION),dev)
	  $(error "Only the `dev` tag can be published from non-master branches")
    endif
  endif
endif
	@echo "==> Publishing $(DOCKER_REPO):$(VERSION)"
	@docker push $(DOCKER_REPO):$(VERSION)
	@echo "==> Your image is now available at $(DOCKER_REPO):$(VERSION)"

.PHONY: vendor
vendor:
	@GO111MODULE=on go mod tidy
	@GO111MODULE=on go mod vendor

.PHONY: clean
clean:
	@echo "==> Cleaning releases"
	@GOOS=${OS} go clean -i -x ./...

debug:
	@echo ${VERSION:v%=%}
