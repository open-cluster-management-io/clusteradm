// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("test clusteradm with addon create", func() {
	ginkgo.BeforeEach(func() {
		ginkgo.By("clear e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("join hub scenario with bootstrap token", func() {
		var err error

		ginkgo.It("should managedclusters join and accepted successfully", func() {
			ginkgo.By("init hub with bootstrap token")
			err = e2e.Clusteradm().Init(
				"--use-bootstrap-token",
				"--context", e2e.Cluster().Hub().Context(),
				"--bundle-version=latest",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			ginkgo.By("managedcluster1 join hub")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", e2e.CommandResult().Token(), "--hub-apiserver", e2e.CommandResult().Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--bundle-version=latest",
				"--force-internal-endpoint-lookup",
				"--singleton",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedcluster1 join error")

			ginkgo.By("hub accept managedcluster1")
			err = e2e.Clusteradm().Accept(
				"--clusters", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm accept error")

			ginkgo.By("hub create addon")
			err = e2e.Clusteradm().Addon(
				"create",
				"test-nginx",
				"-f",
				"scenario/addon/nginx.yaml",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("hub enable addon")
			err = e2e.Clusteradm().Addon(
				"enable",
				"test-nginx",
				"--names",
				"test-nginx",
				"--clusters",
				e2e.Cluster().ManagedCluster1().Name(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("hub disable addon")
			err = e2e.Clusteradm().Addon(
				"disable",
				"test-nginx",
				"--names",
				"test-nginx",
				"--clusters",
				e2e.Cluster().ManagedCluster1().Name(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
})
