// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("test clusteradm with service account", func() {
	ginkgo.BeforeEach(func() {
		e2e.ClearEnv()
	})

	ginkgo.AfterEach(func() {
		ginkgo.By("reset e2e environment...")
		e2e.ResetEnv()
	})

	ginkgo.Context("join hub scenario with service account", func() {

		ginkgo.It("should init hub and accept managed cluster successfully", func() {
			ginkgo.By("clusteradm version check")
			err := e2e.Clusteradm().Version().Run()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm version check error")

			ginkgo.By("init hub")
			jn, err := e2e.Clusteradm().Init(
				"--context", e2e.Cluster().Hub().Context(),
			).Output()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			ginkgo.By("managedcluster1 join hub")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", jn.Token(), "--hub-apiserver", jn.Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--force-internal-endpoint-lookup",
			).Run()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedcluster1 join error")

			ginkgo.By("hub accept managedcluster1")
			err = e2e.Clusteradm().Accept(
				"--clusters", e2e.Cluster().ManagedCluster1().Name(),
				"--wait", "30",
				"--context", e2e.Cluster().Hub().Context(),
			).Run()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm accept error")

			ginkgo.By("get token from hub")

			tk, err := e2e.Clusteradm().Get("token").Output()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm get token error")

			ginkgo.By("managedcluster2 join hub")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster2().Context(),
				"--hub-token", tk.Token(), "--hub-apiserver", tk.Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster2().Name(),
				"--wait",
				"--force-internal-endpoint-lookup",
			).Run()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedcluster2 join error")

			ginkgo.By("hub accept managedcluster2")
			err = e2e.Clusteradm().Accept(
				"--clusters", e2e.Cluster().ManagedCluster2().Name(),
				"--wait", "30",
				"--context", e2e.Cluster().Hub().Context(),
			).Run()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm accept error")

			ginkgo.By("delete token")
			e2e.Clusteradm().Delete(
				"token",
			).Run()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm delete token error")

			ginkgo.By("get token from hub")

			newTk, err := e2e.Clusteradm().Get(
				"token",
			).Output()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm get token error")
			gomega.Expect(newTk.RawCommand()).NotTo(gomega.Equal(tk.RawCommand()), "new token identical as previous token after delete")

		})

	})

})
