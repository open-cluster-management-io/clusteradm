# Copyright Contributors to the Open Cluster Management project

BEFORE_SCRIPT := $(shell build/before-make.sh)

SCRIPTS_PATH ?= build

# Install software dependencies
INSTALL_DEPENDENCIES ?= ${SCRIPTS_PATH}/install-dependencies.sh

GOPATH := ${shell go env GOPATH}
GOOS := ${shell go env GOOS}
GOARCH := ${shell go env GOARCH}

export PROJECT_DIR            = $(shell 'pwd')
export PROJECT_NAME			  = $(shell basename ${PROJECT_DIR})

export GOPACKAGES   = $(shell go list ./... | grep -v /vendor | grep -v /build | grep -v /test )

.PHONY: clean
clean: clean-test clean-e2e

.PHONY: verify
verify:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.1
	go vet ./...
	golangci-lint run --timeout=3m --modules-download-mode vendor -E gofmt ./...

.PHONY: deps
deps:
	@$(INSTALL_DEPENDENCIES)

.PHONY: build
build: 
	rm -f ${GOPATH}/bin/clusteradm
	go install ./cmd/clusteradm/clusteradm.go

.PHONY: 
build-bin:
	@rm -rf bin
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -gcflags=-trimpath=x/y -o bin/clusteradm ./cmd/clusteradm/clusteradm.go && tar -czf bin/clusteradm_darwin_amd64.tar.gz LICENSE -C bin/ clusteradm
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -gcflags=-trimpath=x/y -o bin/clusteradm ./cmd/clusteradm/clusteradm.go && tar -czf bin/clusteradm_darwin_arm64.tar.gz LICENSE -C bin/ clusteradm
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -gcflags=-trimpath=x/y -o bin/clusteradm ./cmd/clusteradm/clusteradm.go && tar -czf bin/clusteradm_linux_amd64.tar.gz LICENSE -C bin/ clusteradm
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -gcflags=-trimpath=x/y -o bin/clusteradm ./cmd/clusteradm/clusteradm.go && tar -czf bin/clusteradm_linux_arm64.tar.gz LICENSE -C bin/ clusteradm
	GOOS=linux GOARCH=ppc64le go build -ldflags="-s -w" -gcflags=-trimpath=x/y -o bin/clusteradm ./cmd/clusteradm/clusteradm.go && tar -czf bin/clusteradm_linux_ppc64le.tar.gz LICENSE -C bin/ clusteradm
	GOOS=linux GOARCH=s390x go build -ldflags="-s -w" -gcflags=-trimpath=x/y -o bin/clusteradm ./cmd/clusteradm/clusteradm.go && tar -czf bin/clusteradm_linux_s390x.tar.gz LICENSE -C bin/ clusteradm
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -gcflags=-trimpath=x/y -o bin/clusteradm.exe ./cmd/clusteradm/clusteradm.go && zip -q bin/clusteradm_windows_amd64.zip LICENSE -j bin/clusteradm.exe

.PHONY: release
release: 
	@if [[ -z "${VERSION}" ]]; then VERSION=`cat VERSION.txt`; echo $$VERSION; fi; \
	git tag $$VERSION && git push upstream --tags

.PHONY: build-krew
build-krew: krew-tools
	@if [[ -z "${VERSION}" ]]; then VERSION=`cat VERSION.txt`; echo $$VERSION; fi; \
	docker run -v ${PROJECT_DIR}/.krew.yaml:/tmp/template-file.yaml rajatjindal/krew-release-bot:v0.0.40 \
	krew-release-bot template --tag v$$VERSION --template-file /tmp/template-file.yaml > krew-manifest.yaml; 
	KREW=/tmp/krew-${GOOS}\_$(GOARCH) && \
	KREW_ROOT=`mktemp -d` KREW_OS=darwin KREW_ARCH=amd64 $$KREW install --manifest=krew-manifest.yaml && \
	KREW_ROOT=`mktemp -d` KREW_OS=linux KREW_ARCH=amd64 $$KREW install --manifest=krew-manifest.yaml && \
	KREW_ROOT=`mktemp -d` KREW_OS=linux KREW_ARCH=arm64 $$KREW install --manifest=krew-manifest.yaml && \
	KREW_ROOT=`mktemp -d` KREW_OS=windows KREW_ARCH=amd64 $$KREW install --manifest=krew-manifest.yaml;

.PHONY: krew-tools
krew-tools:
ifeq (, $(shell which /tmp/krew-$(GOOS)\_$(GOARCH)))
	@( \
		set -x; cd /tmp && \
		KREW=krew-$(GOOS)\_$(GOARCH); \
		curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/$$KREW.tar.gz" && \
		tar zxvf $$KREW.tar.gz \
	) 
endif

.PHONY: install
install: build

.PHONY: plugin
plugin: build
	cp ${GOPATH}/bin/clusteradm ${GOPATH}/bin/oc-clusteradm
	cp ${GOPATH}/bin/clusteradm ${GOPATH}/bin/kubectl-clusteradm

.PHONY: check
## Runs a set of required checks
check: check-copyright

.PHONY: check-copyright
check-copyright:
	@build/check-copyright.sh

.PHONY: test
test:
	@build/run-unit-tests.sh

.PHONY: clean-test
clean-test: 
	-rm -r ./test/unit/coverage
	-rm -r ./test/unit/tmp
	-rm -r ./test/out

include ./test/integration-test.mk
include ./test/e2e/e2e-test.mk

# Update vendor
.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

# Copy CRDs
.PHONY: copy-crd
copy-crd: vendor
	bash -x build/copy-crds.sh
