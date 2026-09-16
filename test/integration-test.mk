# Copyright Contributors to the Open Cluster Management project

export KUBEBUILDER_ASSETS ?=$(LOCAL_BIN)/kubebuilder/bin
export GINKGO ?=$(LOCAL_BIN)/ginkgo

# ref: https://book.kubebuilder.io/reference/envtest.html?highlight=setup-envtest#installation
# Parse the controller-runtime version from go.mod and parse to its release-X.Y git branch
ENVTEST_VERSION ?= $(shell go list -mod=readonly -m -f "{{ .Version }}" sigs.k8s.io/controller-runtime 2>/dev/null | awk -F'[v.]' '{printf "release-%d.%d", $$2, $$3}')
# Parse the Kubernetes API version from go.mod (which is v0.Y.Z) and convert to the corresponding v1.Y.Z format
ENVTEST_K8S_VERSION := $(shell go list -mod=readonly -m -f "{{ .Version }}" k8s.io/api 2>/dev/null | awk -F'[v.]' '{printf "1.%d", $$3}')
ENVTEST := $(LOCAL_BIN)/setup-envtest

envtest-setup:
	# Installing setup-envtest using the release-X.Y branch from the version specified in go.mod
	GOBIN=$(LOCAL_BIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)
.PHONY: envtest-setup

ensure-ginkgo:
	# Downloading ginkgo into '$(GINKGO)'
	GOBIN=$(LOCAL_BIN) go install github.com/onsi/ginkgo/v2/ginkgo@$(shell awk '/github.com\/onsi\/ginkgo\/v2/ {print $$2}' go.mod)
.PHONY: ensure-ginkgo

clean-integration-test:
	$(RM) '$(KB_TOOLS_ARCHIVE_PATH)'
	rm -rf $(TEST_TMP)/kubebuilder
	$(RM) ./integration.test
.PHONY: clean-integration-test

clean: clean-integration-test

test-integration: envtest-setup ensure-ginkgo
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" $(GINKGO) -v ./pkg/cmd/addon/enable  ./pkg/cmd/addon/disable  ./pkg/cmd/install/hubaddon
.PHONY: test-integration
