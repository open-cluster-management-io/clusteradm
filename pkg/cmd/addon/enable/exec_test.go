// Copyright Contributors to the Open Cluster Management project
package enable

import (
	"context"
	"fmt"
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterapiv1 "open-cluster-management.io/api/cluster/v1"
)

var _ = ginkgo.Describe("addon enable", func() {

	// Array of addons to check
	var addons = []string{
		"application-manager",
		"governance-policy-framework",
		"config-policy-controller",
	}

	var (
		cluster1Name string
		cluster2Name string
		suffix       string
		err          error
	)

	ginkgo.BeforeEach(func() {
		suffix = rand.String(5)
		cluster1Name = fmt.Sprintf("cluster-%s", suffix)
		cluster2Name = fmt.Sprintf("cluster-%s", rand.String(5))
	})

	ginkgo.AfterEach(func() {
		ginkgo.By("Delete cluster management add-on")
		for _, addon := range addons {
			err = addonClient.AddonV1alpha1().ClusterManagementAddOns().Delete(
				context.Background(), addon, metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			}
		}
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

	assertCreatingClusterManagementAddOn := func(cmaName string) {
		ginkgo.By(fmt.Sprintf("Create %s ClusterManagementAddOn", cmaName))

		cma := &addonapiv1alpha1.ClusterManagementAddOn{
			ObjectMeta: metav1.ObjectMeta{
				Name: cmaName,
			},
			Spec: addonapiv1alpha1.ClusterManagementAddOnSpec{
				InstallStrategy: addonapiv1alpha1.InstallStrategy{
					Type: addonapiv1alpha1.AddonInstallStrategyManual,
				},
			},
		}

		_, err = addonClient.AddonV1alpha1().ClusterManagementAddOns().Create(
			context.Background(), cma, metav1.CreateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}

	streams := genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

	// Generate entries for the `runWithClient` test table
	addonTests := []ginkgo.TableEntry{}
	for _, addon := range addons {
		addonTests = append(addonTests, ginkgo.Entry(addon, addon))
	}

	ginkgo.DescribeTableSubtree("runWithClient",
		func(addon string) {
			ginkgo.It("Should create ManagedClusterAddOn "+addon+" in each ManagedCluster namespace successfully", func() {
				assertCreatingClusters(cluster1Name)
				assertCreatingClusters(cluster2Name)
				assertCreatingClusterManagementAddOn(addon)

				o := Options{
					Namespace: "open-cluster-management-agent-addon",
					Streams:   streams,
				}

				clusters := []string{cluster1Name, cluster2Name}

				err := o.runWithClient(clusterClient, addonClient, []string{addon}, clusters)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				for _, cluster := range clusters {
					gomega.Eventually(
						addonClient.AddonV1alpha1().ManagedClusterAddOns(cluster).Get,
						eventuallyTimeout, eventuallyInterval,
					).WithArguments(
						context.Background(), addon, metav1.GetOptions{},
					).ShouldNot(gomega.BeNil())
				}
			})
		},
		addonTests,
	)

	ginkgo.Context("runWithClient - invalid configurations", func() {
		ginkgo.It("Should not create a ManagedClusterAddOn because ManagedCluster doesn't exist", func() {
			for _, addon := range addons {
				assertCreatingClusterManagementAddOn(addon)
			}

			clusterName := "no-such-cluster"
			o := Options{
				Streams: streams,
			}

			clusters := []string{clusterName}

			err := o.runWithClient(clusterClient, addonClient, addons, clusters)
			gomega.Expect(err).To(gomega.HaveOccurred())
		})

		ginkgo.It("Should not create a ManagedClusterAddOn because ClusterManagementAddOn doesn't exist", func() {
			assertCreatingClusters(cluster1Name)

			o := Options{
				Namespace: "open-cluster-management-agent-addon",
				Streams:   streams,
			}

			clusters := []string{cluster1Name, cluster1Name, cluster1Name}

			err := o.runWithClient(clusterClient, addonClient, addons, clusters)
			gomega.Expect(err).To(gomega.HaveOccurred())
		})

	})
})
