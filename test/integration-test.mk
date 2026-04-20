# Copyright Contributors to the Open Cluster Management project
TEST_TMP :=/tmp

export KUBEBUILDER_ASSETS ?=$(TEST_TMP)/kubebuilder/bin
export GINKGO ?=$(TEST_TMP)/ginkgo/ginkgo

ENSURE_ENVTEST_SCRIPT := https://raw.githubusercontent.com/open-cluster-management-io/sdk-go/main/ci/envtest/ensure-envtest.sh

.PHONY: envtest-setup
envtest-setup:
	$(eval export KUBEBUILDER_ASSETS=$(shell curl -fsSL $(ENSURE_ENVTEST_SCRIPT) | bash))
	@echo "KUBEBUILDER_ASSETS=$(KUBEBUILDER_ASSETS)"


ensure-ginkgo:
	$(info Downloading ginkgo into '$(TEST_TMP)/ginkgo')
	GOBIN=$(TEST_TMP)/ginkgo go install github.com/onsi/ginkgo/v2/ginkgo@$(shell awk '/github.com\/onsi\/ginkgo\/v2/ {print $$2}' go.mod)
.PHONY: ensure-ginkgo

clean-integration-test:
	$(RM) '$(KB_TOOLS_ARCHIVE_PATH)'
	rm -rf $(TEST_TMP)/kubebuilder
	$(RM) ./integration.test
.PHONY: clean-integration-test

clean: clean-integration-test

test-integration: envtest-setup ensure-ginkgo
	$(GINKGO) -v ./pkg/cmd/addon/enable  ./pkg/cmd/addon/disable  ./pkg/cmd/install/hubaddon 
.PHONY: test-integration
