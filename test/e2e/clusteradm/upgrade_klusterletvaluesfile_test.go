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
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = ginkgo.Describe("test clusteradm upgrade klusterlet with klusterlet values file", ginkgo.Ordered, ginkgo.Label("upgrade-klusterlet-klusterletvaluesfile"), func() {

	ginkgo.BeforeAll(func() {
		ginkgo.By("reset e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	var err error

	ginkgo.Context("upgrade klusterlet with klusterlet values file", func() {
		ginkgo.It("should upgrade klusterlet with configuration from file", func() {
			ginkgo.By("init hub")
			err = e2e.Clusteradm().Init(
				"--context", e2e.Cluster().Hub().Context(),
				"--bundle-version=latest",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			util.WaitClusterManagerApplied(operatorClient)

			ginkgo.By("managedcluster1 join hub with default configuration")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", e2e.CommandResult().Token(),
				"--hub-apiserver", e2e.CommandResult().Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--bundle-version=latest",
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

			mcl1KubeClient, err := kubernetes.NewForConfig(e2e.Cluster().ManagedCluster1().KubeConfig())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("Check the version of operator and agent")
			gomega.Eventually(func() error {
				err := util.CheckOperatorAndAgentVersion(mcl1KubeClient, "latest", "latest")
				if err != nil {
					logf.Log.Error(err, "failed to check operator and agent version")
				}
				return err
			}, 120*time.Second, 5*time.Second).Should(gomega.Succeed())

			ginkgo.By("upgrade klusterlet with advanced klusterlet file")

			klusterletValuesFile, err := filepath.Abs(filepath.Join("testdata", "klusterlet-values.yaml"))
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to get absolute path for klusterlet values file")

			err = e2e.Clusteradm().Upgrade(
				"klusterlet",
				"--bundle-version=latest",
				"--klusterlet-values-file", klusterletValuesFile,
				"--context", e2e.Cluster().ManagedCluster1().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "klusterlet upgrade with klusterlet values file error")

			ginkgo.By("verify klusterlet was upgraded with klusterlet values file")

			mc1OperatorClient, err := operatorclient.NewForConfig(e2e.Cluster().ManagedCluster1().KubeConfig())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				klusterlet, err := mc1OperatorClient.OperatorV1().Klusterlets().Get(context.TODO(), "klusterlet", metav1.GetOptions{})
				if err != nil {
					return err
				}
				if klusterlet.Spec.NodePlacement.NodeSelector == nil {
					return fmt.Errorf("expected node selector to be set from file after upgrade")
				}
				if len(klusterlet.Spec.NodePlacement.Tolerations) == 0 {
					return fmt.Errorf("expected tolerations to be set from file after upgrade")
				}
				if klusterlet.Spec.WorkConfiguration == nil {
					return fmt.Errorf("expected work configuration to be set from file after upgrade")
				}
				foundFeatureGate := false
				for _, featureGate := range klusterlet.Spec.WorkConfiguration.FeatureGates {
					if featureGate.Feature == string(feature.RawFeedbackJsonString) {
						foundFeatureGate = true
					}
				}
				if !foundFeatureGate {
					return fmt.Errorf("expected feature gate %s to be set after upgrade", feature.RawFeedbackJsonString)
				}
				return nil
			}, 120*time.Second, 5*time.Second).Should(gomega.Succeed())
		})
	})
})
