// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"open-cluster-management.io/clusteradm/test/e2e/util"
)

var _ = ginkgo.Describe("test clusteradm with timeout", ginkgo.Label("join-hub-timeout"), func() {
	ginkgo.BeforeEach(func() {
		ginkgo.By("clear e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("join hub scenario with timeout", func() {
		var err error

		ginkgo.It("should managedcluster join and accepted successfully", func() {
			ginkgo.By("init hub with timeout")
			clusterAdm := e2e.Clusteradm()
			err = clusterAdm.Init(
				"--timeout", "400",
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			util.WaitClusterManagerApplied(operatorClient, e2e)

			ginkgo.By("managedcluster1 join hub")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", clusterAdm.Result().Token(), "--hub-apiserver", clusterAdm.Result().Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
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
		})

	})
})
