// Copyright Contributors to the Open Cluster Management project
package disable

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

	"open-cluster-management.io/clusteradm/pkg/cmd/addon/enable"
)

var _ = ginkgo.Describe("addon disable", func() {
	var cluster1Name string
	var cluster2Name string
	var err error

	appMgrAddonName := "application-manager"

	ginkgo.BeforeEach(func() {
		cluster1Name = fmt.Sprintf("cluster-%s", rand.String(5))
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
		gomega.Expect(err).ToNot(gomega.HaveOccurred(), "creat cluster error")
	}

	streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

	assertEnableAddon := func(addons []string, clusters []string, o *enable.Options) {
		ns := o.Namespace

		for _, addon := range addons {
			for _, clus := range clusters {
				ginkgo.By(fmt.Sprintf("Enabling %s addon on %s cluster in %s namespace", addon, clus, ns))

				cai, err := enable.NewClusterAddonInfo(clus, o, addon)
				gomega.Expect(err).ToNot(gomega.HaveOccurred(), "enable addon error")
				err = enable.ApplyAddon(addonClient, cai)
				gomega.Expect(err).ToNot(gomega.HaveOccurred(), "enable addon error")
				fmt.Fprintf(streams.Out, "Deploying %s add-on to namespaces %s of managed cluster: %s.\n", addon, ns, clus)
			}
		}
	}

	ginkgo.Context("runWithClient", func() {

		ginkgo.It("Should disable application-manager ManagedClusterAddOn in ManagedCluster namespace successfully", func() {
			assertCreatingClusters(cluster1Name)

			addons := []string{appMgrAddonName}
			clusters := []string{cluster1Name}
			assertEnableAddon([]string{appMgrAddonName}, []string{cluster1Name}, &enable.Options{Namespace: "default"})

			o := Options{
				Streams: streams,
			}

			err := o.runWithClient(clusterClient, addonClient, kubeClient, apiExtensionsClient, dynamicClient, false, addons, clusters)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.It("Should disable application-manager ManagedClusterAddOns in each ManagedCluster namespace successfully", func() {
			assertCreatingClusters(cluster1Name)
			assertCreatingClusters(cluster2Name)

			addons := []string{appMgrAddonName}
			clusters := []string{cluster1Name, cluster2Name}
			assertEnableAddon(addons, clusters, &enable.Options{Namespace: "default"})

			o := Options{
				Streams: streams,
			}

			err := o.runWithClient(clusterClient, addonClient, kubeClient, apiExtensionsClient, dynamicClient, false, addons, clusters)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.It("Should not disable a ManagedClusterAddOn because ManagedCluster doesn't exist", func() {
			assertCreatingClusters(cluster1Name)

			addons := []string{appMgrAddonName}
			clusters := []string{cluster1Name}
			assertEnableAddon(addons, clusters, &enable.Options{Namespace: "default"})

			wrongCluster := "no-such-addon"
			wrongClusters := []string{wrongCluster}
			o := Options{
				Streams: streams,
			}

			err := o.runWithClient(clusterClient, addonClient, kubeClient, apiExtensionsClient, dynamicClient, false, addons, wrongClusters)
			gomega.Expect(err).To(gomega.HaveOccurred())
		})
	})
})
