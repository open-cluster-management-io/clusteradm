// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/test/e2e/util"
)

var _ = ginkgo.Describe("test clusteradm join with annotations", ginkgo.Label("join-hub-annotate"), func() {
	ginkgo.BeforeEach(func() {
		ginkgo.By("clear e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("join hub scenario with annotations", func() {
		var err error

		ginkgo.It("should managedclusters join with annotations and be accepted successfully", func() {
			ginkgo.By("init hub")
			clusterAdm := e2e.Clusteradm()
			err = clusterAdm.Init(
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			util.WaitClusterManagerApplied(operatorClient, e2e)

			ginkgo.By("managedcluster1 join hub with annotations")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", clusterAdm.Result().Token(), "--hub-apiserver", clusterAdm.Result().Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--force-internal-endpoint-lookup",
				"--klusterlet-annotation", "foo=bar",
				"--klusterlet-annotation", "test=value",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedcluster1 join error")
			gomega.Eventually(func() error {
				return util.ValidateImagePullSecret(managedClusterKubeClient,
					"e30=", "open-cluster-management")
			}, time.Second*120, time.Second*2).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("hub accept managedcluster1")
			err = e2e.Clusteradm().Accept(
				"--clusters", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm accept error")

			ginkgo.By("verify managedcluster1 has correct annotations")
			managedCluster, err := clusterClient.ClusterV1().ManagedClusters().Get(
				context.TODO(), e2e.Cluster().ManagedCluster1().Name(), metav1.GetOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			annotations := managedCluster.GetAnnotations()
			gomega.Expect(annotations).NotTo(gomega.BeNil())
			gomega.Expect(annotations[operatorv1.ClusterAnnotationsKeyPrefix+"/foo"]).To(gomega.Equal("bar"))
			gomega.Expect(annotations[operatorv1.ClusterAnnotationsKeyPrefix+"/test"]).To(gomega.Equal("value"))
		})
	})
})
