ENV ?= "dev"
LOCALDEV_CLUSTER ?= coral

export PATH := ./bin:$(PATH)

CONTROLLER_TOOLS_VERSION ?= v0.16.1
KUSTOMIZE_VERSION ?= v5.4.2
GOLANGCI_LINT_VERSION ?= v2.1.6
GOIMPORTS_VERSION ?= latest
ADDLICENSE_VERSION ?= v1.0.0
BUF_VERSION ?= latest
PROTOC_GEN_GO_VERSION ?= latest
PROTOC_GEN_CONNECT_GO_VERSION ?= latest

KUBECTL ?= kubectl
LOCALBIN ?= $(shell pwd)/bin
TARGETDIR ?= $(shell pwd)/dist
SEACTL_RELEASE_TARGET ?= $(TARGETDIR)/seactl-$(SYSTEM)-$(ARCH).tar.gz
SEACTL_BIN ?= seactl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOIMPORTS = $(LOCALBIN)/goimports
ADDLICENSE = $(LOCALBIN)/addlicense
BUF = $(LOCALBIN)/buf
PROTOC_GEN_GO = $(LOCALBIN)/protoc-gen-go
PROTOC_GEN_CONNECT_GO = $(LOCALBIN)/protoc-gen-connect-go

CRD_OPTIONS ?= "crd:maxDescLen=0,generateEmbeddedObjectMeta=true"
RBAC_OPTIONS ?= "rbac:roleName=coral-system-role"
WEBHOOK_OPTIONS ?= "webhook"
OUTPUT_OPTIONS ?= output:crd:dir=config/coral/crd output:webhook:dir=config/coral/webhook output:rbac:dir=config/coral/rbac
VERSION ?= $(shell git describe --tags --always --dirty)

COVERAGE ?= 1
ifeq ($(COVERAGE), 1)
	GO_COVERPROFILE = "-coverprofile=cover.out"
else
	GO_COVERPROFILE = ""
endif

VERBOSE ?= 0
ifeq ($(VERBOSE), 1)
	GO_VERBOSE = "-v"
else
	GO_VERBOSE =
endif

###
### Generators
###
.PHONY: codegen
codegen: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./pkg/apis/..."

.PHONY: manifests
manifests: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) paths="./pkg/..." $(CRD_OPTIONS) $(RBAC_OPTIONS) $(WEBHOOK_OPTIONS) $(OUTPUT_OPTIONS)

.PHONY: generate
generate: codegen manifests

###
### Build, install, run, and clean
###
.PHONY: install
install: $(KUSTOMIZE) deps generate
	@$(KUSTOMIZE) build config/coral/crd | kubectl apply -f -
	@$(KUSTOMIZE) build config/coral/overlays/$(ENV) | kubectl apply -f -

.PHONY: uninstall
uninstall:
	@kubectl delete -k config/overlays/$(ENV)

###
### Tests/Utils
###
.PHONY: test
test:
	go test ./... $(GO_VERBOSE) $(GO_COVERPROFILE)

.PHONY: lint
lint: $(GOLANGCI_LINT)
	@$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: $(GOLANGCI_LINT) $(GOIMPORTS)
	@$(GOLANGCI_LINT) run --fix
	@$(GOIMPORTS) -w .

.PHONY: fmt
fmt:
	@$(GOBIN) fmt ./...

.PHONY: vet
vet:
	@$(GOBIN) vet ./...

.PHONY: license
license: $(ADDLICENSE)
	@find . -name '*.go' | xargs $(ADDLICENSE) -c "Coral Authors" -y 2025 -l apache

###
### Builds
###
$(TARGETDIR):
	mkdir -p $(TARGETDIR)

.PHONY: build
build: $(TARGETDIR)
	CGO_ENABLED=0 go build -trimpath --ldflags "-s -w -X ctx.sh/coral/pkg/build.Version=$(VERSION)" -o $(TARGETDIR)/coral ./pkg/cmd/coral

###
### Individual dep installs were copied out of kubebuilder testdata makefiles.
###
deps: $(CONTROLLER_GEN) $(KUSTOMIZE) $(GOLANGCI_LINT) $(ADDLICENSE) $(BUF) $(PROTOC_GEN_GO) $(PROTOC_GEN_CONNECT_GO)

$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

$(CONTROLLER_GEN): $(LOCALBIN)
	@test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

$(KUSTOMIZE):
	@test -s $(KUSTOMIZE) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

$(GOLANGCI_LINT): $(LOCALBIN)
	@test -s $(GOLANGCI_LINT) || \
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}

$(GOIMPORTS): $(LOCALBIN)
	@test -s $(GOIMPORTS) || \
	GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@${GOIMPORTS_VERSION}

$(ADDLICENSE): $(LOCALBIN)
	@test -s $(ADDLICENSE) || \
  GOBIN=$(LOCALBIN) go install github.com/google/addlicense@$(ADDLICENSE_VERSION)

$(BUF): $(LOCALBIN)
	@test -s $(BUF) || \
	GOBIN=$(LOCALBIN) go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)

$(PROTOC_GEN_GO): $(LOCALBIN)
	@test -s $(PROTOC_GEN_GO) || \
	GOBIN=$(LOCALBIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)

$(PROTOC_GEN_CONNECT_GO): $(LOCALBIN)
	@test -s $(PROTOC_GEN_CONNECT_GO) || \
	GOBIN=$(LOCALBIN) go install connectrpc.com/connect/cmd/protoc-gen-connect-go@$(PROTOC_GEN_CONNECT_GO_VERSION)

.PHONY: clean
clean:
	@k3d cluster delete coral

###
### Local development
###
.PHONY: localdev
localdev: localdev-cluster localdev-shared install

.PHONY: localdev-cluster
localdev-cluster:
	@if k3d cluster get $(LOCALDEV_CLUSTER) --no-headers >/dev/null 2>&1;  \
		then echo "Cluster exists, skipping creation"; \
		else k3d cluster create --config config/k3d/config.yaml --volume $(PWD):/app; \
		fi

.PHONY: localdev-shared
localdev-shared:
	@$(KUSTOMIZE) build config/cert-manager | envsubst | $(KUBECTL) apply -f -
	@$(KUBECTL) wait --for=condition=available --timeout=120s deploy -l app.kubernetes.io/group=cert-manager -n cert-manager
	@$(KUSTOMIZE) build config/localstack | envsubst | $(KUBECTL) apply -f -
	@$(KUBECTL) wait --for=condition=available --timeout=120s deploy/localstack -n coral-system
	@$(KUSTOMIZE) build config/registry | envsubst | $(KUBECTL) apply -f -
	@$(KUBECTL) wait --for=condition=available --timeout=120s deploy/registry -n coral-system
	@$(KUSTOMIZE) build config/coral/overlays/$(ENV) | envsubst | $(KUBECTL) apply -f -
	@$(KUBECTL) wait --for=condition=available --timeout=120s deploy/coral-controller -n coral-system

.PHONY: localdev-clean
localdev-clean:
#	@$(KUBECTL) delete -k config/registry
#	@$(KUBECTL) delete -k config/cert-manager
	@$(KUBECTL) delete -k config/coral/overlays/$(ENV)

.PHONY: controller-run
controller-run:
	$(eval POD := $(shell kubectl get pods -n coral-system -l app=controller -o=custom-columns=:metadata.name --no-headers))
	@$(KUBECTL) exec -n coral-system -it pod/$(POD) -- bash -c "go run pkg/cmd/coral/*.go controller --log-level=5 --skip-insecure-verify=true"

.PHONY: controller-exec
controller-exec:
	$(eval POD := $(shell kubectl get pods -n coral-system -l app=controller -o=custom-columns=:metadata.name --no-headers))
	@$(KUBECTL) exec -n coral-system -it pod/$(POD) -- bash

.PHONY: agent-run
agent-run:
	$(eval POD := $(shell kubectl get pods -n coral-system -l app=agent -o=custom-columns=:metadata.name --no-headers))
	@$(KUBECTL) exec -n coral-system -it pod/$(POD) -- bash -c "go run pkg/cmd/coral/*.go agent --log-level=5"

.PHONY: agent-exec
agent-exec:
	$(eval POD := $(shell kubectl get pods -n coral-system -l app=agent -o=custom-columns=:metadata.name --no-headers))
	@$(KUBECTL) exec -n coral-system -it pod/$(POD) -- bash
