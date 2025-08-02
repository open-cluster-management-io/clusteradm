// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	feature "open-cluster-management.io/api/feature"
	"open-cluster-management.io/clusteradm/test/e2e/util"
)

var _ = ginkgo.Describe("test clusteradm join with klusterlet file", ginkgo.Label("join-hub-klusterletfile"), func() {
	ginkgo.BeforeEach(func() {
		ginkgo.By("clear e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("join hub scenario with advanced klusterlet file", func() {
		var err error

		ginkgo.It("should join with advanced klusterlet configuration from file", func() {
			ginkgo.By("init hub")
			err = e2e.Clusteradm().Init(
				"--context", e2e.Cluster().Hub().Context(),
				"--bundle-version=latest",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			util.WaitClusterManagerApplied(operatorClient)

			ginkgo.By("managedcluster1 join hub with advanced klusterlet file")

			klusterletFile, err := filepath.Abs(filepath.Join("testdata", "klusterlet-advanced.yaml"))
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to get absolute path for klusterlet file")

			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", e2e.CommandResult().Token(),
				"--hub-apiserver", e2e.CommandResult().Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--klusterlet-file", klusterletFile,
				"--wait",
				"--bundle-version=latest",
				"--force-internal-endpoint-lookup",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedcluster1 join with advanced klusterlet file error")

			ginkgo.By("hub accept managedcluster1")
			err = e2e.Clusteradm().Accept(
				"--clusters", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm accept error")

			mcl1KubeClient, err := kubernetes.NewForConfig(e2e.Cluster().ManagedCluster1().KubeConfig())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			mc1OperatorClient, err := operatorclient.NewForConfig(e2e.Cluster().ManagedCluster1().KubeConfig())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("verify klusterlet was created with advanced configuration from file")
			gomega.Eventually(func() error {
				klusterlet, err := mc1OperatorClient.OperatorV1().Klusterlets().Get(context.TODO(), "klusterlet", metav1.GetOptions{})
				if err != nil {
					return err
				}

				// Verify the namespace from the klusterlet file is used
				expectedNamespace := "open-cluster-management-agent-advanced"
				if klusterlet.Spec.Namespace != expectedNamespace {
					return fmt.Errorf("expected namespace %s, got %s", expectedNamespace, klusterlet.Spec.Namespace)
				}

				// Verify node placement from the klusterlet file is used
				if klusterlet.Spec.NodePlacement.NodeSelector == nil {
					return fmt.Errorf("expected node selector to be set from file")
				}
				if len(klusterlet.Spec.NodePlacement.Tolerations) == 0 {
					return fmt.Errorf("expected tolerations to be set from file")
				}

				// Verify work configuration from the klusterlet file is used
				if klusterlet.Spec.WorkConfiguration == nil {
					return fmt.Errorf("expected work configuration to be set from file after join")
				}
				foundFeatureGate := false
				for _, featureGate := range klusterlet.Spec.WorkConfiguration.FeatureGates {
					if featureGate.Feature == string(feature.RawFeedbackJsonString) {
						foundFeatureGate = true
					}
				}
				if !foundFeatureGate {
					return fmt.Errorf("expected feature gate %s to be set after join", feature.RawFeedbackJsonString)
				}

				return nil
			}, 120*time.Second, 5*time.Second).Should(gomega.Succeed())

			ginkgo.By("verify klusterlet namespace was created with the custom name")
			gomega.Eventually(func() error {
				_, err := mcl1KubeClient.CoreV1().Namespaces().Get(context.TODO(), "open-cluster-management-agent-advanced", metav1.GetOptions{})
				return err
			}, 120*time.Second, 5*time.Second).Should(gomega.Succeed())
		})
	})
})
