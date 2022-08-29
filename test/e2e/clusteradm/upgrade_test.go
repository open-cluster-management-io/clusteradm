// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("test clusteradm upgrade clustermanager and Klusterlets", ginkgo.Ordered, func() {

	ginkgo.BeforeAll(func() {
		ginkgo.By("reset e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		err = e2e.ResetEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	var err error

	ginkgo.It("run cluster manager upgrade version latest ", func() {
		ginkgo.Skip("Upgrade is skipped due to flaky when destroying control plane. Need to revisit it after fix cleanup issue")
		err = e2e.Clusteradm().Upgrade(
			"clustermanager",
			"--bundle-version", "latest",
			"--context", e2e.Cluster().Hub().Context(),
		)

		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm upgrade error")

		err = e2e.Clusteradm().Upgrade(
			"klusterlet",
			"--bundle-version", "latest",
			"--context", e2e.Cluster().ManagedCluster1().Context(),
		)

		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "klusterlet upgrade error")
	})
})
