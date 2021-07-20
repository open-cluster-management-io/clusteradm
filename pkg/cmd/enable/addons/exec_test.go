// Copyright Contributors to the Open Cluster Management project
package addons

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	clusterapiv1 "open-cluster-management.io/api/cluster/v1"
)

var _ = ginkgo.Describe("enable addons", func() {
	var cluster1Name string
	var cluster2Name string
	var suffix string
	var err error

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

	ginkgo.Context("runWithClient", func() {
		ginkgo.It("Should create an application-manager ManagedClusterAddOn in ManagedCluster namespace successfully", func() {
			assertCreatingClusters(cluster1Name)

			o := Options{
				values: Values{
					clusters: []string{cluster1Name},
					addons:   []string{appMgrAddonName},
				},
			}

			err := o.runWithClient(clusterClient, kubeClient, apiExtensionsClient, dynamicClient, false)
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
				values: Values{
					clusters: []string{cluster1Name, cluster2Name},
					addons:   []string{appMgrAddonName},
				},
			}

			err := o.runWithClient(clusterClient, kubeClient, apiExtensionsClient, dynamicClient, false)
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
				values: Values{
					clusters: []string{clusterName},
					addons:   []string{appMgrAddonName},
				},
			}

			err := o.runWithClient(clusterClient, kubeClient, apiExtensionsClient, dynamicClient, false)
			gomega.Expect(err).To(gomega.HaveOccurred())
		})

		ginkgo.It("Should not create a ManagedClusterAddOn because it's not a valid add-on name", func() {
			assertCreatingClusters(cluster1Name)

			addonName := "no-such-addon"
			o := Options{
				values: Values{
					clusters: []string{cluster1Name},
					addons:   []string{addonName},
				},
			}

			err := o.runWithClient(clusterClient, kubeClient, apiExtensionsClient, dynamicClient, false)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Consistently(func() error {
				_, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(cluster1Name).Get(context.Background(), addonName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, consistentlyTimeout, consistentlyInterval).Should(gomega.HaveOccurred())
		})
	})
})
