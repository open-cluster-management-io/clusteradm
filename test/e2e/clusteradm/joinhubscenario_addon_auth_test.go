// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/test/e2e/util"
)

var _ = ginkgo.Describe("test clusteradm join with addon-kubeclient-registration-auth", ginkgo.Label("join-hub-addon-auth"), func() {
	ginkgo.BeforeEach(func() {
		ginkgo.By("clear e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("join hub scenario with CSR addon authentication", func() {
		ginkgo.It("should managedclusters join and accepted successfully with CSR addon auth", func() {
			ginkgo.By("init hub")
			clusterAdm := e2e.Clusteradm()
			err := clusterAdm.Init(
				"--context", e2e.Cluster().Hub().Context(),
				"--bundle-version", bundleVersion,
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			util.WaitClusterManagerApplied(operatorClient, e2e)

			ginkgo.By("join hub with CSR addon auth (default)")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", clusterAdm.Result().Token(),
				"--hub-apiserver", clusterAdm.Result().Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--addon-kubeclient-registration-auth", "csr",
				"--wait",
				"--force-internal-endpoint-lookup",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedCluster join error")

			ginkgo.By("accept the managedclusters")
			err = e2e.Clusteradm().Accept(
				"--context", e2e.Cluster().Hub().Context(),
				"--clusters", e2e.Cluster().ManagedCluster1().Name(),
				"--wait", "60",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("verify addon kubeclient registration driver is not set (using default CSR)")
			// Create operator client for managed cluster
			mcl1OperatorClient, err := operatorclient.NewForConfig(e2e.Cluster().ManagedCluster1().KubeConfig())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			var klusterlet *operatorv1.Klusterlet
			gomega.Eventually(func() error {
				klusterlet, err = mcl1OperatorClient.OperatorV1().Klusterlets().Get(
					context.TODO(), "klusterlet", metav1.GetOptions{})
				return err
			}, time.Second*60, time.Second*2).Should(gomega.Succeed())

			gomega.Expect(klusterlet.Spec.RegistrationConfiguration).NotTo(gomega.BeNil())
			// When using CSR (default), AddOnKubeClientRegistrationDriver should not be set
			gomega.Expect(klusterlet.Spec.RegistrationConfiguration.AddOnKubeClientRegistrationDriver).To(gomega.BeNil())
		})
	})

	ginkgo.Context("join hub scenario with token addon authentication", func() {
		ginkgo.It("should managedclusters join and accepted successfully with token addon auth", func() {
			ginkgo.By("init hub")
			clusterAdm := e2e.Clusteradm()
			err := clusterAdm.Init(
				"--context", e2e.Cluster().Hub().Context(),
				"--bundle-version", bundleVersion,
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			util.WaitClusterManagerApplied(operatorClient, e2e)

			ginkgo.By("join hub with token addon auth")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", clusterAdm.Result().Token(),
				"--hub-apiserver", clusterAdm.Result().Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--addon-kubeclient-registration-auth", "token",
				"--addon-token-expiration-seconds", "3600",
				"--wait",
				"--force-internal-endpoint-lookup",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedCluster join error")

			ginkgo.By("accept the managedclusters")
			err = e2e.Clusteradm().Accept(
				"--context", e2e.Cluster().Hub().Context(),
				"--clusters", e2e.Cluster().ManagedCluster1().Name(),
				"--wait", "60",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("verify addon kubeclient registration driver is set to token")
			// Create operator client for managed cluster
			mcl1OperatorClient, err := operatorclient.NewForConfig(e2e.Cluster().ManagedCluster1().KubeConfig())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			var klusterlet *operatorv1.Klusterlet
			gomega.Eventually(func() error {
				klusterlet, err = mcl1OperatorClient.OperatorV1().Klusterlets().Get(
					context.TODO(), "klusterlet", metav1.GetOptions{})
				return err
			}, time.Second*60, time.Second*2).Should(gomega.Succeed())

			gomega.Expect(klusterlet.Spec.RegistrationConfiguration).NotTo(gomega.BeNil())
			gomega.Expect(klusterlet.Spec.RegistrationConfiguration.AddOnKubeClientRegistrationDriver).NotTo(gomega.BeNil())
			gomega.Expect(klusterlet.Spec.RegistrationConfiguration.AddOnKubeClientRegistrationDriver.AuthType).To(
				gomega.Equal("token"))
			gomega.Expect(klusterlet.Spec.RegistrationConfiguration.AddOnKubeClientRegistrationDriver.Token).NotTo(gomega.BeNil())
			gomega.Expect(klusterlet.Spec.RegistrationConfiguration.AddOnKubeClientRegistrationDriver.Token.ExpirationSeconds).To(
				gomega.Equal(int64(3600)))
		})
	})

	ginkgo.Context("join hub scenario with token addon authentication and default expiration", func() {
		ginkgo.It("should managedclusters join with token auth and default expiration", func() {
			ginkgo.By("init hub")
			clusterAdm := e2e.Clusteradm()
			err := clusterAdm.Init(
				"--context", e2e.Cluster().Hub().Context(),
				"--bundle-version", bundleVersion,
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			util.WaitClusterManagerApplied(operatorClient, e2e)

			ginkgo.By("join hub with token addon auth and default expiration (0)")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", clusterAdm.Result().Token(),
				"--hub-apiserver", clusterAdm.Result().Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--addon-kubeclient-registration-auth", "token",
				"--wait",
				"--force-internal-endpoint-lookup",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedCluster join error")

			ginkgo.By("accept the managedclusters")
			err = e2e.Clusteradm().Accept(
				"--context", e2e.Cluster().Hub().Context(),
				"--clusters", e2e.Cluster().ManagedCluster1().Name(),
				"--wait", "60",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("verify addon kubeclient registration driver has token auth with nil Token config")
			// Create operator client for managed cluster
			mcl1OperatorClient, err := operatorclient.NewForConfig(e2e.Cluster().ManagedCluster1().KubeConfig())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			var klusterlet *operatorv1.Klusterlet
			gomega.Eventually(func() error {
				klusterlet, err = mcl1OperatorClient.OperatorV1().Klusterlets().Get(
					context.TODO(), "klusterlet", metav1.GetOptions{})
				return err
			}, time.Second*60, time.Second*2).Should(gomega.Succeed())

			gomega.Expect(klusterlet.Spec.RegistrationConfiguration).NotTo(gomega.BeNil())
			gomega.Expect(klusterlet.Spec.RegistrationConfiguration.AddOnKubeClientRegistrationDriver).NotTo(gomega.BeNil())
			gomega.Expect(klusterlet.Spec.RegistrationConfiguration.AddOnKubeClientRegistrationDriver.AuthType).To(
				gomega.Equal("token"))
			// When expiration is 0 (default), Token config should not be set
			gomega.Expect(klusterlet.Spec.RegistrationConfiguration.AddOnKubeClientRegistrationDriver.Token).To(gomega.BeNil())
		})
	})
})
