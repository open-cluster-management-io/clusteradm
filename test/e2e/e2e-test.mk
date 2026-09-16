# Copyright Contributors to the Open Cluster Management project
export KUBECONFIG := ${HOME}/.kube/config

export HUB_NAME := ${PROJECT_NAME}-e2e-test-hub
export MANAGED_CLUSTER1_NAME := ${PROJECT_NAME}-e2e-test-c1

export HUB_CTX := kind-${HUB_NAME}
export MANAGED_CLUSTER1_CTX := kind-${MANAGED_CLUSTER1_NAME}

BUNDLE_VERSION?=latest

clean-e2e: 
	kind delete cluster --name ${HUB_NAME}
	kind delete cluster --name ${MANAGED_CLUSTER1_NAME}
.PHONY: clean-e2e

# start clusters and set context variables
start-cluster: 
	kind create cluster --name ${MANAGED_CLUSTER1_NAME}
	kind create cluster --name ${HUB_NAME} --image kindest/node:v1.35.8@sha256:07b2536e30b803ed61d1677a79df6115f798ce64c80f9e22f6ed45afd09323c0
.PHONY: start-cluster

test-e2e: clean-e2e ensure-ginkgo start-cluster deps install test-e2e-only
.PHONY: test-e2e

test-e2e-only:
	$(GINKGO) -v --timeout 3600s $(if $(GINKGO_LABEL_FILTER),--label-filter="$(GINKGO_LABEL_FILTER)") ./test/e2e/clusteradm -- --bundle-version=$(BUNDLE_VERSION)
.PHONY: test-e2e-only
