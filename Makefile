# Copyright Contributors to the Open Cluster Management project

BEFORE_SCRIPT := $(shell build/before-make.sh)

SCRIPTS_PATH ?= build

# Install software dependencies
INSTALL_DEPENDENCIES ?= ${SCRIPTS_PATH}/install-dependencies.sh

GOPATH := ${shell go env GOPATH}

export PROJECT_DIR            = $(shell 'pwd')
export PROJECT_NAME			  = $(shell basename ${PROJECT_DIR})

export GOPACKAGES   = $(shell go list ./... | grep -v /vendor | grep -v /build | grep -v /test )

.PHONY: clean
clean: clean-test
	kind delete cluster --name ${PROJECT_NAME}-functional-test-hub
	kind delete cluster --name ${PROJECT_NAME}-functional-test-c1
	kind delete cluster --name ${PROJECT_NAME}-functional-test-c2
	
.PHONY: deps
deps:
	@$(INSTALL_DEPENDENCIES)

.PHONY: build
build: 
	rm -f ${GOPATH}/bin/clusteradm
	go install ./cmd/clusteradm.go

.PHONY: 
build-bin:
	@rm -rf bin
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -gcflags=-trimpath=x/y -o bin/clusteradm_darwin_amd64 ./cmd/clusteradm.go && tar -czf bin/clusteradm_darwin_amd64.tar.gz -C bin/ clusteradm_darwin_amd64 
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -gcflags=-trimpath=x/y -o bin/clusteradm_linux_amd64 ./cmd/clusteradm.go && tar -czf bin/clusteradm_linux_amd64.tar.gz -C bin/ clusteradm_linux_amd64 
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -gcflags=-trimpath=x/y -o bin/clusteradm_linux_arm64 ./cmd/clusteradm.go && tar -czf bin/clusteradm_linux_arm64.tar.gz -C bin/ clusteradm_linux_arm64 
	GOOS=linux GOARCH=ppc64le go build -ldflags="-s -w" -gcflags=-trimpath=x/y -o bin/clusteradm_linux_ppc64le ./cmd/clusteradm.go && tar -czf bin/clusteradm_linux_ppc64le.tar.gz -C bin/ clusteradm_linux_ppc64le 
	GOOS=linux GOARCH=s390x go build -ldflags="-s -w" -gcflags=-trimpath=x/y -o bin/clusteradm_linux_s390x ./cmd/clusteradm.go && tar -czf bin/clusteradm_linux_s390x.tar.gz -C bin/ clusteradm_linux_s390x 
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -gcflags=-trimpath=x/y -o bin/clusteradm_windows_amd64.exe ./cmd/clusteradm.go && zip -q bin/clusteradm_windows_amd64.zip -j bin/clusteradm_windows_amd64.exe

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
	-rm -r ./test/functional/tmp
	-rm -r ./test/out

.PHONY: functional-test-full
functional-test-full: deps install
	@build/run-functional-tests.sh

include ./test/integration-test.mk
