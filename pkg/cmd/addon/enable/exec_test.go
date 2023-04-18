// Copyright Contributors to the Open Cluster Management project
package enable

import (
	"context"
	"fmt"
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	clusterapiv1 "open-cluster-management.io/api/cluster/v1"
)

var _ = ginkgo.Describe("addon enable", func() {
	var cluster1Name string
	var cluster2Name string
	var suffix string
	var err error

	appMgrAddonName := "application-manager"

	ginkgo.BeforeEach(func() {
		suffix = rand.String(5)
		cluster1Name = fmt.Sprintf("cluster-%s", suffix)
		cluster2Name = fmt.Sprintf("cluster-%s", rand.String(5))
	})

	assertCreatingClusters := func(clusterName string) {
		ginkgo.By(fmt.Sprintf("Create %s cluster", clusterName))

		cluster := &clusterapiv1.ManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterName,
			},
		}

		_, err = clusterClient.ClusterV1().ManagedClusters().Create(context.Background(), cluster, metav1.CreateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterName,
			},
		}
		_, err := kubeClient.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}

	streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

	ginkgo.Context("runWithClient", func() {
		ginkgo.It("Should create an application-manager ManagedClusterAddOn in ManagedCluster namespace successfully", func() {
			assertCreatingClusters(cluster1Name)

			o := Options{
				Namespace: "open-cluster-management-agent-addon",
				Streams:   streams,
			}

			addons := []string{appMgrAddonName}
			clusters := []string{cluster1Name, cluster1Name, cluster1Name}

			err := o.runWithClient(clusterClient, addonClient, addons, clusters)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(cluster1Name).Get(context.Background(), appMgrAddonName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("Should create application-manager ManagedClusterAddOns in each ManagedCluster namespace successfully", func() {
			assertCreatingClusters(cluster1Name)
			assertCreatingClusters(cluster2Name)

			o := Options{
				Namespace: "open-cluster-management-agent-addon",
				Streams:   streams,
			}

			addons := []string{appMgrAddonName}
			clusters := []string{cluster1Name, cluster2Name, cluster1Name}

			err := o.runWithClient(clusterClient, addonClient, addons, clusters)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(cluster1Name).Get(context.Background(), appMgrAddonName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(cluster2Name).Get(context.Background(), appMgrAddonName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("Should not create a ManagedClusterAddOn because ManagedCluster doesn't exist", func() {
			clusterName := "no-such-cluster"
			o := Options{
				Streams: streams,
			}

			addons := []string{appMgrAddonName}
			clusters := []string{clusterName}

			err := o.runWithClient(clusterClient, addonClient, addons, clusters)
			gomega.Expect(err).To(gomega.HaveOccurred())
		})

	})
})
