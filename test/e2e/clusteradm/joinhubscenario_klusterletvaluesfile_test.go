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

var _ = ginkgo.Describe("test clusteradm join with klusterlet values file", ginkgo.Label("join-hub-klusterletvaluesfile"), func() {
	ginkgo.BeforeEach(func() {
		ginkgo.By("clear e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("join hub scenario with klusterlet values file", func() {
		var err error

		ginkgo.It("should join with klusterlet chart values file", func() {
			ginkgo.By("init hub")
			clusterAdm := e2e.Clusteradm()
			err = clusterAdm.Init(
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			util.WaitClusterManagerApplied(operatorClient, e2e)

			ginkgo.By("managedcluster1 join hub with klusterlet values file")

			klusterletValuesFile, err := filepath.Abs(filepath.Join("test", "e2e", "clusteradm", "testdata", "klusterlet-values-join.yaml"))
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to get absolute path for klusterlet values file")

			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", clusterAdm.Result().Token(),
				"--hub-apiserver", clusterAdm.Result().Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--klusterlet-values-file", klusterletValuesFile,
				"--wait",
				"--force-internal-endpoint-lookup",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedcluster1 join with klusterlet values file error")

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

			ginkgo.By("verify klusterlet was created with klusterlet values file")
			gomega.Eventually(func() error {
				klusterlet, err := mc1OperatorClient.OperatorV1().Klusterlets().Get(context.TODO(), "klusterlet", metav1.GetOptions{})
				if err != nil {
					return err
				}
				expectedNamespace := "open-cluster-management-agent-advanced"
				if klusterlet.Spec.Namespace != expectedNamespace {
					return fmt.Errorf("expected namespace %s, got %s", expectedNamespace, klusterlet.Spec.Namespace)
				}
				if klusterlet.Spec.NodePlacement.NodeSelector == nil {
					return fmt.Errorf("expected node selector to be set from file")
				}
				if len(klusterlet.Spec.NodePlacement.Tolerations) == 0 {
					return fmt.Errorf("expected tolerations to be set from file")
				}
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
