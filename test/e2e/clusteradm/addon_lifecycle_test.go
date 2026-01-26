// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"open-cluster-management.io/clusteradm/test/e2e/util"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var addonLabels = map[string]string{
	"foo.example.com/created-by": "clusteradm",
	"foo":                        "bar",
}

var _ = ginkgo.Describe("test clusteradm with addon create", ginkgo.Label("addon-create"), func() {
	ginkgo.BeforeEach(func() {
		ginkgo.By("clear e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("create template type addon", func() {
		var err error

		ginkgo.It("should managedclusters join and accepted successfully", func() {
			ginkgo.By("init hub with bootstrap token")
			clusterAdm := e2e.Clusteradm()
			err = clusterAdm.Init(
				"--use-bootstrap-token",
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

			util.WaitClusterManagerApplied(operatorClient, e2e)

			ginkgo.By("managedcluster1 join hub")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", clusterAdm.Result().Token(), "--hub-apiserver", clusterAdm.Result().Host(),
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--force-internal-endpoint-lookup",
				"--singleton",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedcluster1 join error")

			ginkgo.By("hub accept managedcluster1")
			err = e2e.Clusteradm().Accept(
				"--clusters", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm accept error")

			ginkgo.By("create configmap-reader clusterrole")
			_, err = kubeClient.RbacV1().ClusterRoles().Create(context.TODO(), &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "configmap-reader",
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get", "list", "watch"},
						APIGroups: []string{""},
						Resources: []string{"configmaps"},
					},
				},
			}, metav1.CreateOptions{})
			if !errors.IsAlreadyExists(err) {
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), "create configmap-reader clusterrole error")
			}

			var pairs []string
			for k, v := range addonLabels {
				pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
			}
			labelsString := strings.Join(pairs, ",")

			ginkgo.By("hub create addon")
			err = e2e.Clusteradm().Addon(
				"create",
				"test-nginx",
				"-f",
				"test/e2e/clusteradm/scenario/addon/nginx.yaml",
				"--hub-registration",
				"--cluster-role-bind",
				"configmap-reader",
				"--labels",
				labelsString,
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				cma, err := addonClient.AddonV1alpha1().ClusterManagementAddOns().Get(
					context.TODO(), "test-nginx", metav1.GetOptions{})
				if err != nil {
					return err
				}
				if !reflect.DeepEqual(cma.Labels, addonLabels) {
					return fmt.Errorf("clusterManagementAddOns does not have expected labels. have: %v, want: %v", cma.Labels, addonLabels)
				}

				addonT, err := addonClient.AddonV1alpha1().AddOnTemplates().Get(
					context.TODO(), "test-nginx-0.0.1", metav1.GetOptions{})
				if err != nil {
					return err
				}
				if !reflect.DeepEqual(addonT.Labels, addonLabels) {
					return fmt.Errorf("addOnTemplate does not have expected labels. have: %v, want: %v", addonT.Labels, addonLabels)
				}

				return nil
			}, 120*time.Second, 1*time.Second).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("hub enable addon")
			err = e2e.Clusteradm().Addon(
				"enable",
				"test-nginx",
				"--names",
				"test-nginx",
				"--clusters",
				e2e.Cluster().ManagedCluster1().Name(),
				"--labels",
				labelsString,
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				mca, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(e2e.Cluster().ManagedCluster1().Name()).Get(
					context.TODO(), "test-nginx", metav1.GetOptions{})
				if err != nil {
					return err
				}
				if !reflect.DeepEqual(mca.Labels, addonLabels) {
					return fmt.Errorf("managedClusterAddOn does not have expected labels. have: %v, want: %v", mca.Labels, addonLabels)
				}

				if meta.IsStatusConditionTrue(mca.Status.Conditions, "Available") {
					return nil
				}
				return fmt.Errorf("addon is not available: %v", mca.Status.Conditions)
			}, 300*time.Second, 1*time.Second).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("hub disable addon")
			err = e2e.Clusteradm().Addon(
				"disable",
				"test-nginx",
				"--names",
				"test-nginx",
				"--clusters",
				e2e.Cluster().ManagedCluster1().Name(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// check the addon was deleted, otherwise the next test CleanEnv may fail due to
			// the applied manifest work is not deleted
			ginkgo.By("check the addon was deleted")
			gomega.Eventually(func() error {
				mws, err := dynamicClient.Resource(schema.GroupVersionResource{
					Group:    "work.open-cluster-management.io",
					Version:  "v1",
					Resource: "manifestworks",
				}).Namespace(e2e.Cluster().ManagedCluster1().Name()).List(context.TODO(),
					metav1.ListOptions{
						LabelSelector: "open-cluster-management.io/addon-name=test-nginx",
					})
				if err != nil {
					return err
				}
				if len(mws.Items) == 0 {
					return nil
				}
				return fmt.Errorf("addon test-nginx manifestwork is not deleted yet")
			}, 60*time.Second, 1*time.Second).ShouldNot(gomega.HaveOccurred())
		})
	})
})
