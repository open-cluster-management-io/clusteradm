// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("test clusteradm with bootstrap token", func() {
	ginkgo.BeforeEach(func() {
		e2e.ClearEnv()
	})

	ginkgo.AfterEach(func() {
		ginkgo.By("reset e2e environment...")
		e2e.ResetEnv()
	})

	ginkgo.Context("join hub scenario with bootstrap token", func() {
		var originalToken string
		var err error

		ginkgo.It("should managedclusters join and accepted successfully", func() {
			ginkgo.By("init hub with bootstrap token")
			err = e2e.Clusteradm().Init(
				"--use-bootstrap-token",
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			ginkgo.By("managedcluster1 join hub")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", e2e.CommandResult().Token(), "--hub-apiserver", e2e.CommandResult().Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--force-internal-endpoint-lookup",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedcluster1 join error")

			ginkgo.By("hub accept managedcluster1")
			err = e2e.Clusteradm().Accept(
				"--clusters", e2e.Cluster().ManagedCluster1().Name(),
				"--wait", "30",
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm accept error")

			ginkgo.By("get token from hub")
			err = e2e.Clusteradm().Get(
				"token",
				"--use-bootstrap-token",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm get token error")

			originalToken = e2e.CommandResult().RawCommand()

			ginkgo.By("managedcluster2 join hub")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster2().Context(),
				"--hub-token", e2e.CommandResult().Token(), "--hub-apiserver", e2e.CommandResult().Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster2().Name(),
				"--wait",
				"--force-internal-endpoint-lookup",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedcluster2 join error")

			ginkgo.By("hub accept managedcluster2")
			err = e2e.Clusteradm().Accept(
				"--clusters", e2e.Cluster().ManagedCluster2().Name(),
				"--wait", "30",
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm accept error")

			ginkgo.By("delete token")
			err = e2e.Clusteradm().Delete(
				"token",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm delete token error")

			ginkgo.By("get token from hub")
			err = e2e.Clusteradm().Get(
				"token",
				"--use-bootstrap-token",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm get token error")

			gomega.Expect(e2e.CommandResult().RawCommand()).NotTo(gomega.Equal(originalToken), "new token identical as previous token after delete")
		})

	})
})
