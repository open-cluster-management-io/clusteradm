// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"open-cluster-management.io/clusteradm/pkg/version"
	"open-cluster-management.io/clusteradm/test/e2e/util"
)

var _ = ginkgo.Describe("test clusteradm upgrade clustermanager and klusterlets", ginkgo.Ordered, ginkgo.Label("upgrade"), func() {

	ginkgo.BeforeAll(func() {
		ginkgo.By("reset e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	var err error

	ginkgo.It("run cluster manager and klusterlet upgrade version latest ", func() {
		ginkgo.By("init hub with service account")
		clusteradmDefault := e2e.Clusteradm().WithVersion("default")
		err = clusteradmDefault.Init(
			"--context", e2e.Cluster().Hub().Context(),
		)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

		ginkgo.By("Check the version of operator and controller")
		gomega.Eventually(func() error {
			return util.CheckOperatorAndManagerVersion(kubeClient, version.GetDefaultBundleVersion(), version.GetDefaultBundleVersion())
		}, 120*time.Second, 5*time.Second).Should(gomega.Succeed())

		ginkgo.By("managedcluster1 join hub")
		err = clusteradmDefault.Join(
			"--context", e2e.Cluster().ManagedCluster1().Context(),
			"--hub-token", clusteradmDefault.Result().Token(), "--hub-apiserver", clusteradmDefault.Result().Host(),
			"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
			"--wait",
			"--force-internal-endpoint-lookup",
		)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedcluster1 join error")

		ginkgo.By("hub accept managedcluster1")
		err = clusteradmDefault.Accept(
			"--clusters", e2e.Cluster().ManagedCluster1().Name(),
			"--wait",
			"--context", e2e.Cluster().Hub().Context(),
		)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm accept error")

		mcl1KubeClient, err := kubernetes.NewForConfig(e2e.Cluster().ManagedCluster1().KubeConfig())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		ginkgo.By("Check the version of operator and agent")
		gomega.Eventually(func() error {
			return util.CheckOperatorAndAgentVersion(mcl1KubeClient, bundleVersion, version.GetDefaultBundleVersion())
		}, 120*time.Second, 5*time.Second).Should(gomega.Succeed())

		err = e2e.Clusteradm().Upgrade(
			"clustermanager",
			"--context", e2e.Cluster().Hub().Context(),
			"--wait",
		)

		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm upgrade error")

		ginkgo.By("Upgrade to the latest version")
		gomega.Eventually(func() error {
			return util.CheckOperatorAndManagerVersion(kubeClient, bundleVersion, bundleVersion)
		}, 120*time.Second, 5*time.Second).Should(gomega.Succeed())

		err = e2e.Clusteradm().Upgrade(
			"klusterlet",
			"--context", e2e.Cluster().ManagedCluster1().Context(),
			"--wait",
		)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "klusterlet upgrade error")

		gomega.Eventually(func() error {
			return util.CheckOperatorAndAgentVersion(mcl1KubeClient, bundleVersion, bundleVersion)
		}, 120*time.Second, 5*time.Second).Should(gomega.Succeed())
	})
})
