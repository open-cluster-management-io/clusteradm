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
				"--bundle-version=latest",
				"--registration-drivers=awsirsa",
				"--hub-cluster-arn=arn:aws:eks:us-west-2:123456789012:cluster/hub-cluster1",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			cm, err = operatorClient.OperatorV1().ClusterManagers().Get(context.TODO(), "cluster-manager", metav1.GetOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			// Ensure that when only awsirsa is passed as registration-drivers only awsirsa driver is available
			gomega.Expect(len(cm.Spec.RegistrationConfiguration.RegistrationDrivers)).Should(gomega.Equal(1))
			gomega.Expect(cm.Spec.RegistrationConfiguration.RegistrationDrivers[0].AuthType).Should(gomega.Equal("awsirsa"))

			err = e2e.Clusteradm().Init(
				"--use-bootstrap-token",
				"--context", e2e.Cluster().Hub().Context(),
				"--bundle-version=latest",
				"--registration-drivers=awsirsa,csr",
				"--hub-cluster-arn=arn:aws:eks:us-west-2:123456789012:cluster/hub-cluster1",
				"--aws-resource-tags=product:v1:tenant:app-name=My-App,product:v1:tenant:created-by=Team-1",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			cm, err = operatorClient.OperatorV1().ClusterManagers().Get(context.TODO(), "cluster-manager", metav1.GetOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			// Ensure that awsirsa and csr is passed as registration-drivers both the values are set.
			gomega.Expect(len(cm.Spec.RegistrationConfiguration.RegistrationDrivers)).Should(gomega.Equal(2))

			gomega.Expect(cm.Spec.RegistrationConfiguration.RegistrationDrivers[0].AuthType).Should(gomega.Equal("awsirsa"))
			gomega.Expect(cm.Spec.RegistrationConfiguration.RegistrationDrivers[0].AwsIrsa.HubClusterArn).
				Should(gomega.Equal("arn:aws:eks:us-west-2:123456789012:cluster/hub-cluster1"))
			gomega.Expect(cm.Spec.RegistrationConfiguration.RegistrationDrivers[0].AwsIrsa.Tags).
				Should(gomega.Equal("product:v1:tenant:app-name=My-App,product:v1:tenant:created-by=Team-1"))

			gomega.Expect(cm.Spec.RegistrationConfiguration.RegistrationDrivers[1].AuthType).Should(gomega.Equal("csr"))

			err = e2e.Clusteradm().Init(
				"--use-bootstrap-token",
				"--context", e2e.Cluster().Hub().Context(),
				"--bundle-version=latest",
				"--registration-drivers=awsirsa,csr",
				"--hub-cluster-arn=arn:aws:eks:us-west-2:123456789012:cluster/hub-cluster1",
				"--feature-gates=ManagedClusterAutoApproval=true",
				"--auto-approved-csr-identities=user1,user2",
				"--auto-approved-arn-patterns=arn:aws:eks:us-west-2:123456789013:cluster/.*,arn:aws:eks:us-west-2:123456789012:cluster/.*",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")
			cm, err = operatorClient.OperatorV1().ClusterManagers().Get(context.TODO(), "cluster-manager", metav1.GetOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			// Ensure that the auto approval identities contain user for CSR and pattern for AWS
			gomega.Expect(cm.Spec.RegistrationConfiguration.RegistrationDrivers[0].AuthType).Should(gomega.Equal("awsirsa"))
			gomega.Expect(cm.Spec.RegistrationConfiguration.RegistrationDrivers[0].AwsIrsa.HubClusterArn).
				Should(gomega.Equal("arn:aws:eks:us-west-2:123456789012:cluster/hub-cluster1"))
			gomega.Expect(cm.Spec.RegistrationConfiguration.RegistrationDrivers[0].AwsIrsa.AutoApprovedIdentities).
				Should(gomega.Equal("arn:aws:eks:us-west-2:123456789013:cluster/.*,arn:aws:eks:us-west-2:123456789012:cluster/.*"))

			gomega.Expect(cm.Spec.RegistrationConfiguration.RegistrationDrivers[1].AuthType).Should(gomega.Equal("csr"))
			gomega.Expect(cm.Spec.RegistrationConfiguration.RegistrationDrivers[1].CSR.AutoApprovedIdentities).
				Should(gomega.Equal("user1,user2"))

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
