// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"open-cluster-management.io/clusteradm/test/e2e/util"
)

var e2e *util.TestE2eConfig

func TestE2EClusteradm(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "E2E clusteradm test")
}

var _ = ginkgo.BeforeSuite(func() {
	ginkgo.By("Starting e2e test environment")

	// set cluster info and start clusters.
	e2e = util.PrepareE2eEnvironment()
})
