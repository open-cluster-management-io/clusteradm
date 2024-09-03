// Copyright Contributors to the Open Cluster Management project

package clusteradme2e

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"open-cluster-management.io/clusteradm/test/e2e/util"
)

var _ = ginkgo.Describe("test clusteradm with bootstrap token in singleton mode", func() {
	ginkgo.BeforeEach(func() {
		ginkgo.By("clear e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("init cluster manager", func() {

		ginkgo.It("should init multiple times with different flags", func() {
			ginkgo.By("init hub with bootstrap token")
			err := e2e.Clusteradm().Init(
				"--use-bootstrap-token",
				"--context", e2e.Cluster().Hub().Context(),
				"--bundle-version=latest",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			cm, err := operatorClient.OperatorV1().ClusterManagers().Get(context.TODO(), "cluster-manager", metav1.GetOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(len(cm.Spec.RegistrationConfiguration.FeatureGates)).Should(gomega.Equal(1))

			err = e2e.Clusteradm().Init(
				"--use-bootstrap-token",
				"--context", e2e.Cluster().Hub().Context(),
				"--feature-gates=ManagedClusterAutoApproval=true",
				"--bundle-version=latest",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")
			cm, err = operatorClient.OperatorV1().ClusterManagers().Get(context.TODO(), "cluster-manager", metav1.GetOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(len(cm.Spec.RegistrationConfiguration.FeatureGates)).Should(gomega.Equal(2))

			gomega.Eventually(func() error {
				return util.ValidateImagePullSecret(kubeClient, "e30=",
					"open-cluster-management")
			}, time.Second*120, time.Second*2).ShouldNot(gomega.HaveOccurred())

			// set image-pull-credential
			encodedString := util.NewTestImagePullCredentialFile("config.json")
			err = e2e.Clusteradm().Init(
				"--use-bootstrap-token",
				"--context", e2e.Cluster().Hub().Context(),
				"--feature-gates=ManagedClusterAutoApproval=true",
				"--bundle-version=latest",
				"--image-pull-credential-file=./config.json",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			util.CleanupTestImagePullCredentialFile("config.json")
			gomega.Eventually(func() error {
				return util.ValidateImagePullSecret(kubeClient, encodedString,
					"open-cluster-management")
			}, time.Second*120, time.Second*2).ShouldNot(gomega.HaveOccurred())
		})
	})
})
