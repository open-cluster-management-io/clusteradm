// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapiv1 "open-cluster-management.io/api/cluster/v1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/test/e2e/util"
)

var _ = ginkgo.Describe("test clusteradm join with grpc", ginkgo.Label("join-hub-grpc"), func() {
	ginkgo.BeforeEach(func() {
		ginkgo.By("clear e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})
	ginkgo.AfterEach(func() {
		err := util.WaitClustersDeleted(e2e.Cluster().Hub().KubeConfig())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = e2e.Clusteradm().Unjoin(
			"--context", e2e.Cluster().Hub().Context(),
			"--cluster-name", e2e.Cluster().Hub().Name(),
		)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("join hub scenario with grpc", func() {
		var err error

		ginkgo.It("should managedCluster join with grpc", func() {
			ginkgo.By("init hub")
			err = e2e.Clusteradm().Init(
				"--context", e2e.Cluster().Hub().Context(),
				"--bundle-version=latest",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			ginkgo.By("wait for cluster-manager CR to be created and update cluster-manager to enable grpc")
			var clusterManager *operatorv1.ClusterManager
			gomega.Eventually(func() error {
				clusterManager, err = operatorClient.OperatorV1().ClusterManagers().Get(
					context.TODO(), "cluster-manager", metav1.GetOptions{})
				if err != nil {
					return err
				}

				// Enable ManagedClusterAutoApproval feature gate
				if clusterManager.Spec.RegistrationConfiguration == nil {
					clusterManager.Spec.RegistrationConfiguration = &operatorv1.RegistrationHubConfiguration{}
				}
				clusterManager.Spec.RegistrationConfiguration.FeatureGates = append(
					clusterManager.Spec.RegistrationConfiguration.FeatureGates,
					operatorv1.FeatureGate{
						Feature: "ManagedClusterAutoApproval",
						Mode:    operatorv1.FeatureGateModeTypeEnable,
					},
				)

				// Add grpc authType in registrationDrivers
				clusterManager.Spec.RegistrationConfiguration.RegistrationDrivers = []operatorv1.RegistrationDriverHub{
					{
						AuthType: operatorv1.GRPCAuthType,
						GRPC: &operatorv1.GRPCRegistrationConfig{
							AutoApprovedIdentities: []string{
								"system:serviceaccount:open-cluster-management:agent-registration-bootstrap",
							},
						},
					},
				}

				// Add serverConfiguration with grpc protocol
				clusterManager.Spec.ServerConfiguration = &operatorv1.ServerConfiguration{
					EndpointsExposure: []operatorv1.EndpointExposure{
						{
							Protocol: "grpc",
						},
					},
				}

				_, err = operatorClient.OperatorV1().ClusterManagers().Update(
					context.TODO(), clusterManager, metav1.UpdateOptions{})
				return err
			}, time.Second*30, time.Second*2).Should(gomega.Succeed())

			util.WaitClusterManagerApplied(operatorClient)

			ginkgo.By(fmt.Sprintf("join hub as managedCluster %s with grpc", e2e.Cluster().Hub().Name()))
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().Hub().Context(),
				"--hub-token", e2e.CommandResult().Token(),
				"--hub-apiserver", e2e.CommandResult().Host(),
				"--registration-auth", operatorv1.GRPCAuthType,
				"--grpc-server", "cluster-manager-grpc-server.open-cluster-management-hub.svc:8090",
				"--cluster-name", e2e.Cluster().Hub().Name(),
				"--bundle-version=latest",
				"--force-internal-endpoint-lookup",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedCluster join error")

			ginkgo.By("wait for klusterlet CR to be created and verify grpc authType")

			var klusterlet *operatorv1.Klusterlet
			gomega.Eventually(func() error {
				klusterlet, err = operatorClient.OperatorV1().Klusterlets().Get(
					context.TODO(), "klusterlet", metav1.GetOptions{})
				return err
			}, time.Second*60, time.Second*2).Should(gomega.Succeed())

			ginkgo.By("verify klusterlet has grpc authType in registrationConfiguration")
			gomega.Expect(klusterlet.Spec.RegistrationConfiguration).NotTo(gomega.BeNil())
			gomega.Expect(klusterlet.Spec.RegistrationConfiguration.RegistrationDriver).NotTo(gomega.BeNil())
			gomega.Expect(klusterlet.Spec.RegistrationConfiguration.RegistrationDriver.AuthType).To(
				gomega.Equal(operatorv1.GRPCAuthType), "klusterlet should have grpc authType")

			ginkgo.By(fmt.Sprintf("wait for cluster %s to become available", e2e.Cluster().Hub().Name()))
			gomega.Eventually(func() bool {
				managedCluster, err := clusterClient.ClusterV1().ManagedClusters().Get(
					context.TODO(), e2e.Cluster().Hub().Name(), metav1.GetOptions{})
				if err != nil {
					return false
				}
				// Check if ManagedClusterConditionAvailable is true
				for _, condition := range managedCluster.Status.Conditions {
					if condition.Type == clusterapiv1.ManagedClusterConditionAvailable && condition.Status == metav1.ConditionTrue {
						return true
					}
				}
				return false
			}, time.Second*120, time.Second*2).Should(gomega.BeTrue(), "managedCluster should become available")
		})
	})
})
