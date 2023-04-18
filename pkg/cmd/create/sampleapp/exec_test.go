// Copyright Contributors to the Open Cluster Management project
package sampleapp

import (
	"context"
	"fmt"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	clusterapiv1 "open-cluster-management.io/api/cluster/v1"
	"open-cluster-management.io/clusteradm/pkg/cmd/addon/enable"
	installScenario "open-cluster-management.io/clusteradm/pkg/cmd/install/hubaddon/scenario"
)

// AddonOptions: options used for addon deployment
type AddonOptions struct {
	values Values
}

// Values: The values used in the addons deployment template
type Values struct {
	hubAddons []string
}

var _ = ginkgo.Describe("deploy samepleapp to every managed cluster", func() {
	var cluster1Name string
	var cluster2Name string
	var err error

	const (
		testSampleAppName     = "sampleapp"
		testNamespace         = "default"
		appMgrAddonName       = "application-manager"
		installAddonDir       = "install/hubaddon/scenario/addon/appmgr"
		enableAddonFile       = "addons/addon.yaml"
		installAddonNamespace = "open-cluster-management"
		enableAddonNamespace  = "default"
		dryRun                = false
		clusterSetLabel       = "cluster.open-cluster-management.io/clusterset"
		placementLabel        = "placement"
		placementLabelValue   = "sampleapp"
		channelGroup          = "apps.open-cluster-management.io"
		channelVersion        = "v1"
		channelResource       = "channels"
		subscriptionGroup     = "apps.open-cluster-management.io"
		subscriptionVersion   = "v1"
		subscriptionresource  = "subscriptions"
	)

	ginkgo.BeforeEach(func() {
		cluster1Name = fmt.Sprintf("cluster-%s", rand.String(5))
		cluster2Name = fmt.Sprintf("cluster-%s", rand.String(5))
	})

	assertCreatingClusters := func(clusterName string) {
		ginkgo.By(fmt.Sprintf("Create %s cluster", clusterName))

		cluster := &clusterapiv1.ManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterName,
				Labels: map[string]string{
					clusterSetLabel: fmt.Sprintf("app-%s", testSampleAppName),
					placementLabel:  placementLabelValue,
				},
			},
		}

		_, err = clusterClient.ClusterV1().ManagedClusters().Create(context.Background(), cluster, metav1.CreateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterName,
			},
		}
		_, err = kubeClient.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}

	streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

	addonPathWalkDir := func(root string) ([]string, error) {
		var files []string
		err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				relPath, err := filepath.Rel(filepath.Dir(filepath.Dir(root)), path)
				if err != nil {
					return err
				}
				files = append(files, relPath)
			}
			return nil
		})
		return files, err
	}

	contains := func(objlists *unstructured.UnstructuredList, item string) bool {
		for _, obj := range objlists.Items {
			if obj.GetName() == item {
				return true
			}
		}
		return false
	}

	assertInstallAddon := func(addon string, addonNamespace string, addonDir string) {

		ginkgo.By(fmt.Sprintf("installing %s addon", addon))

		ao := AddonOptions{
			values: Values{
				hubAddons: []string{addon},
			},
		}

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: addonNamespace,
			},
		}

		_, err = kubeClient.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		f := cmdutil.NewFactory(TestClientGetter{cfg: restConfig})
		r := reader.NewResourceReader(f.NewBuilder(), false, genericclioptions.IOStreams{Out: os.Stdout, ErrOut: os.Stderr})

		mydir, err := os.Getwd()
		gomega.Expect(err).ToNot(gomega.HaveOccurred(), "install addon error")
		appDir := filepath.Join(filepath.Dir(filepath.Dir(mydir)), addonDir)

		files, err := addonPathWalkDir(appDir)
		gomega.Expect(err).ToNot(gomega.HaveOccurred(), "install addon error")

		err = r.Apply(installScenario.Files, ao.values, files...)
		gomega.Expect(err).ToNot(gomega.HaveOccurred(), "install addon error")

		fmt.Fprintf(streams.Out, "Installing built-in %s add-on to namespace %s.\n", addon, addonNamespace)
	}

	assertEnableAddon := func(addon string, clusters []string, addonNamespace string, addonFilePath string) {
		for _, clus := range clusters {

			ginkgo.By(fmt.Sprintf("Enabling %s addon on %s cluster in %s namespce", addon, clus, addonNamespace))

			eo := enable.Options{
				Namespace: addonNamespace,
			}

			cai, err := enable.NewClusterAddonInfo(clus, &eo, addon)
			gomega.Expect(err).ToNot(gomega.HaveOccurred(), "enable addon error")

			err = enable.ApplyAddon(addonClient, cai)
			gomega.Expect(err).ToNot(gomega.HaveOccurred(), "enable addon error")

			fmt.Fprintf(streams.Out, "Deploying %s add-on to namespace %s of managed cluster: %s.\n", addon, addonNamespace, clus)
		}
	}

	ginkgo.Context("runWithClient", func() {

		ginkgo.It("Should deploy a sample application to all managed clusters", func() {
			assertCreatingClusters(cluster1Name)
			assertCreatingClusters(cluster2Name)

			o := Options{
				Streams:       streams,
				SampleAppName: testSampleAppName,
				Namespace:     testNamespace,
			}

			clusters := []string{cluster1Name, cluster2Name}

			assertInstallAddon(appMgrAddonName, installAddonNamespace, installAddonDir)
			assertEnableAddon(appMgrAddonName, clusters, enableAddonNamespace, enableAddonFile)

			err = o.runWithClient(clusterClient, dryRun)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {

				var (
					placementResourceName        = fmt.Sprintf("%s-placement", o.SampleAppName)
					managedClusterSetName        = fmt.Sprintf("app-%s", o.SampleAppName)
					managedClusterSetBindingName = fmt.Sprintf("app-%s", o.SampleAppName)
					channelName                  = fmt.Sprintf("%s-helmrepo", o.SampleAppName)
					subscriptionName             = fmt.Sprintf("%s-subscription", o.SampleAppName)
				)

				if _, err = clusterClient.ClusterV1beta1().Placements(testNamespace).Get(context.TODO(), placementResourceName, metav1.GetOptions{}); err != nil {
					return fmt.Errorf(fmt.Sprintf("Missing Placement resource \"%s\" in namespace %s", placementResourceName, testNamespace))
				}
				fmt.Fprintf(streams.Out, "Placement resource \"%s\" created successfully in namespace %s.\n", placementResourceName, testNamespace)

				if _, err = clusterClient.ClusterV1beta2().ManagedClusterSets().Get(context.TODO(), managedClusterSetName, metav1.GetOptions{}); err != nil {
					return fmt.Errorf(fmt.Sprintf("Missing ManagedClusterSet resource \"%s\"", managedClusterSetName))
				}
				fmt.Fprintf(streams.Out, "ManagedClusterSet resource \"%s\" created successfully.\n", managedClusterSetName)

				if _, err = clusterClient.ClusterV1beta2().ManagedClusterSetBindings(testNamespace).Get(context.TODO(), managedClusterSetBindingName, metav1.GetOptions{}); err != nil {
					return fmt.Errorf(fmt.Sprintf("Missing ManagedClusterSetBinding resource \"%s\" in namespace %s", managedClusterSetBindingName, testNamespace))
				}
				fmt.Fprintf(streams.Out, "ManagedClusterSetBinding resource \"%s\" created successfully in namespace %s.\n", managedClusterSetBindingName, testNamespace)

				channelGVR := schema.GroupVersionResource{
					Group:    channelGroup,
					Version:  channelVersion,
					Resource: channelResource,
				}

				channelObjlist, _ := dynamicClient.Resource(channelGVR).List(context.TODO(), metav1.ListOptions{})
				if !contains(channelObjlist, channelName) {
					return fmt.Errorf(fmt.Sprintf("Missing Channel custom resource \"%s\"", channelName))
				}
				fmt.Fprintf(streams.Out, "Channel custom resource \"%s\" created successfully in namespace %s.\n", channelName, testNamespace)

				subscriptionGVR := schema.GroupVersionResource{
					Group:    subscriptionGroup,
					Version:  subscriptionVersion,
					Resource: subscriptionresource,
				}

				subscriptionObjlist, _ := dynamicClient.Resource(subscriptionGVR).List(context.TODO(), metav1.ListOptions{})
				if !contains(subscriptionObjlist, subscriptionName) {
					return fmt.Errorf(fmt.Sprintf("Missing Subscription custom resource \"%s\"", subscriptionName))
				}
				fmt.Fprintf(streams.Out, "Subscription custom resource \"%s\" created successfully in namespace %s.\n", subscriptionName, testNamespace)

				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})
	})
})
