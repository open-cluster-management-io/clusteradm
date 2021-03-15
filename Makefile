BEFORE_SCRIPT := $(shell build/before-make.sh)

SCRIPTS_PATH ?= build

# Install software dependencies
INSTALL_DEPENDENCIES ?= ${SCRIPTS_PATH}/install-dependencies.sh

export GOPACKAGES   = $(shell go list ./... | grep -v /vendor | grep -v /build | grep -v /test )

.PHONY: deps
deps:
	$(INSTALL_DEPENDENCIES)

.PHONY: build
build: go-bindata
	go install ./cmd/cm.go

.PHONY: check
## Runs a set of required checks
check: go-bindata-check lint

.PHONY: go-bindata
go-bindata:
	$(GOPATH)/bin/go-bindata -nometadata -pkg bindata -o pkg/bindata/bindata_generated.go -prefix resources/  resources/...

.PHONY: go-bindata-check
go-bindata-check:
	@if which go-bindata > /dev/null; then \
		echo "##### Updating go-bindata..."; \
		cd $(mktemp -d) && GOSUMDB=off go get -u github.com/go-bindata/go-bindata/...; \
	else \
		echo "##### installing go-bindata..."; \
		cd $(mktemp -d) && GOSUMDB=off go get -u github.com/go-bindata/go-bindata/...; \
	fi
	@$(GOPATH)/bin/go-bindata --version
	@echo "##### go-bindata-check ####"
	@$(GOPATH)/bin/go-bindata -nometadata -pkg bindata -o $(BINDATA_TEMP_DIR)/bindata_generated.go -prefix resources/  resources/...; \
	diff $(BINDATA_TEMP_DIR)/bindata_generated.go pkg/bindata/bindata_generated.go > go-bindata.diff; \
	if [ $$? != 0 ]; then \
	  echo "#### Difference detected and saved in go-bindata.diff, run 'make go-bindata' to regenerate the bindata_generated.go"; \
	  cat go-bindata.diff; \
	  exit 1; \
	fi
	@echo "##### go-bindata-check #### Success"

.PHONY: test
test: go-bindata
	@build/run-unit-tests.sh