// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("test clusteradm upgrade clustermanager and Klusterlets", ginkgo.Ordered, func() {

	ginkgo.BeforeAll(func() {
		ginkgo.By("reset e2e environment...")
		e2e.ClearEnv()
		e2e.ResetEnv()
	})

	var err error

	ginkgo.It("run cluster manager upgrade version 0.8.0 ", func() {
		err = e2e.Clusteradm().Upgrade(
			"clustermanager",
			"--bundle-version", "0.8.0",
			"--context", e2e.Cluster().Hub().Context(),
		)
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm upgrade error")

	ginkgo.It("run klusterlet upgrade version 0.8.0 ", func() {
		err = e2e.Clusteradm().Upgrade(
			"klusterlet",
			"--bundle-version", "0.8.0",
			"--context", e2e.Cluster().ManagedCluster1().Context(),
		)
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "klusterlet upgrade error")
})
