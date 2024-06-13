// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"open-cluster-management.io/clusteradm/test/e2e/util"
)

var _ = ginkgo.Describe("test clusteradm with bootstrap token in singleton mode", func() {
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

			gomega.Eventually(func() error {
				return util.ValidateImagePullSecret(managedClusterKubeClient,
					"e30K", "open-cluster-management")
			}, time.Second*120, time.Second*2).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("hub accept managedcluster1")
			err = e2e.Clusteradm().Accept(
				"--clusters", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm accept error")
		})

	})
})
