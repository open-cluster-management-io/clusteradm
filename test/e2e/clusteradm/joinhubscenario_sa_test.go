// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"open-cluster-management.io/clusteradm/test/e2e/util"
)

var _ = ginkgo.Describe("test clusteradm with service account", func() {
	ginkgo.BeforeEach(func() {
		ginkgo.By("clear e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("join hub scenario with service account", func() {
		var originalToken string
		var err error

		ginkgo.It("should managedclusters join and accepted successfully", func() {
			ginkgo.By("init hub with service account")
			err = e2e.Clusteradm().Init(
				"--context", e2e.Cluster().Hub().Context(),
				"--bundle-version=latest",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			util.WaitClusterManagerApplied(operatorClient)

			ginkgo.By("managedcluster1 join hub")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", e2e.CommandResult().Token(), "--hub-apiserver", e2e.CommandResult().Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--bundle-version=latest",
				"--force-internal-endpoint-lookup",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedcluster1 join error")

			ginkgo.By("hub accept managedcluster1")
			err = e2e.Clusteradm().Accept(
				"--clusters", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm accept error")

			ginkgo.By("get token from hub")
			err = e2e.Clusteradm().Get("token")
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm get token error")

			originalToken = e2e.CommandResult().RawCommand()

			ginkgo.By("delete token")
			err = e2e.Clusteradm().Delete(
				"token",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm delete token error")

			ginkgo.By("get token from hub")
			err = e2e.Clusteradm().Get(
				"token",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm get token error")

			gomega.Expect(e2e.CommandResult().RawCommand()).NotTo(gomega.Equal(originalToken), "new token identical as previous token after delete")
		})

	})

})
