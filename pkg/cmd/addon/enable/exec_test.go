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
		"argocd",
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

	ginkgo.Context("addon enable with config file", func() {
		var (
			configNamespace string
			configName      string
		)

		ginkgo.BeforeEach(func() {
			configNamespace = fmt.Sprintf("config-ns-%s", rand.String(5))
			configName = fmt.Sprintf("config-%s", rand.String(5))

			// Create namespace for config
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: configNamespace,
				},
			}
			_, err := kubeClient.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.AfterEach(func() {
			// Clean up AddOnDeploymentConfig
			err := addonClient.AddonV1alpha1().AddOnDeploymentConfigs(configNamespace).Delete(
				context.Background(), configName, metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			}

			// Clean up namespace
			err = kubeClient.CoreV1().Namespaces().Delete(
				context.Background(), configNamespace, metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			}
		})

		ginkgo.It("Should create ManagedClusterAddOn with configs from file", func() {
			assertCreatingClusters(cluster1Name)
			assertCreatingClusterManagementAddOn("argocd")

			// Create a config file
			configContent := fmt.Sprintf(`apiVersion: addon.open-cluster-management.io/v1alpha1
kind: AddOnDeploymentConfig
metadata:
  name: %s
  namespace: %s
spec:
  customizedVariables:
  - name: LOG_LEVEL
    value: "debug"
`, configName, configNamespace)

			tmpFile, err := os.CreateTemp("", "addon-config-*.yaml")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(configContent)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			tmpFile.Close()

			o := Options{
				Namespace:  "open-cluster-management-agent-addon",
				ConfigFile: tmpFile.Name(),
				Streams:    streams,
			}

			// Set up factory for the options
			o.ClusteradmFlags = testFlags

			clusters := []string{cluster1Name}

			err = o.runWithClient(clusterClient, addonClient, []string{"argocd"}, clusters)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			// Verify ManagedClusterAddOn was created with configs
			mca, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(cluster1Name).Get(
				context.Background(), "argocd", metav1.GetOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(mca.Spec.Configs).To(gomega.HaveLen(1))
			gomega.Expect(mca.Spec.Configs[0].Group).To(gomega.Equal("addon.open-cluster-management.io"))
			gomega.Expect(mca.Spec.Configs[0].Resource).To(gomega.Equal("addondeploymentconfigs"))
			gomega.Expect(mca.Spec.Configs[0].Namespace).To(gomega.Equal(configNamespace))
			gomega.Expect(mca.Spec.Configs[0].Name).To(gomega.Equal(configName))

			// Verify AddOnDeploymentConfig was created
			adc, err := addonClient.AddonV1alpha1().AddOnDeploymentConfigs(configNamespace).Get(
				context.Background(), configName, metav1.GetOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(adc.Name).To(gomega.Equal(configName))
		})

		ginkgo.It("Should create ManagedClusterAddOn with multiple configs from file", func() {
			assertCreatingClusters(cluster1Name)
			assertCreatingClusterManagementAddOn("governance-policy-framework")

			config1Name := fmt.Sprintf("config1-%s", rand.String(5))
			config2Name := fmt.Sprintf("config2-%s", rand.String(5))

			// Create a config file with multiple configs
			configContent := fmt.Sprintf(`apiVersion: addon.open-cluster-management.io/v1alpha1
kind: AddOnDeploymentConfig
metadata:
  name: %s
  namespace: %s
spec:
  customizedVariables:
  - name: LOG_LEVEL
    value: "debug"
---
apiVersion: addon.open-cluster-management.io/v1alpha1
kind: AddOnDeploymentConfig
metadata:
  name: %s
  namespace: %s
spec:
  customizedVariables:
  - name: REPLICAS
    value: "3"
`, config1Name, configNamespace, config2Name, configNamespace)

			tmpFile, err := os.CreateTemp("", "addon-config-*.yaml")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(configContent)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			tmpFile.Close()

			o := Options{
				Namespace:  "open-cluster-management-agent-addon",
				ConfigFile: tmpFile.Name(),
				Streams:    streams,
			}

			// Set up factory for the options
			o.ClusteradmFlags = testFlags

			clusters := []string{cluster1Name}

			err = o.runWithClient(clusterClient, addonClient, []string{"governance-policy-framework"}, clusters)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			// Verify ManagedClusterAddOn was created with multiple configs
			mca, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(cluster1Name).Get(
				context.Background(), "governance-policy-framework", metav1.GetOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(mca.Spec.Configs).To(gomega.HaveLen(2))

			// Check first config
			gomega.Expect(mca.Spec.Configs[0].Group).To(gomega.Equal("addon.open-cluster-management.io"))
			gomega.Expect(mca.Spec.Configs[0].Resource).To(gomega.Equal("addondeploymentconfigs"))
			gomega.Expect(mca.Spec.Configs[0].Namespace).To(gomega.Equal(configNamespace))
			gomega.Expect(mca.Spec.Configs[0].Name).To(gomega.Equal(config1Name))

			// Check second config
			gomega.Expect(mca.Spec.Configs[1].Group).To(gomega.Equal("addon.open-cluster-management.io"))
			gomega.Expect(mca.Spec.Configs[1].Resource).To(gomega.Equal("addondeploymentconfigs"))
			gomega.Expect(mca.Spec.Configs[1].Namespace).To(gomega.Equal(configNamespace))
			gomega.Expect(mca.Spec.Configs[1].Name).To(gomega.Equal(config2Name))

			// Verify both AddOnDeploymentConfigs were created
			_, err = addonClient.AddonV1alpha1().AddOnDeploymentConfigs(configNamespace).Get(
				context.Background(), config1Name, metav1.GetOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			_, err = addonClient.AddonV1alpha1().AddOnDeploymentConfigs(configNamespace).Get(
				context.Background(), config2Name, metav1.GetOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			// Clean up additional configs
			err = addonClient.AddonV1alpha1().AddOnDeploymentConfigs(configNamespace).Delete(
				context.Background(), config1Name, metav1.DeleteOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			err = addonClient.AddonV1alpha1().AddOnDeploymentConfigs(configNamespace).Delete(
				context.Background(), config2Name, metav1.DeleteOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.It("Should work without config file when not provided", func() {
			assertCreatingClusters(cluster1Name)
			assertCreatingClusterManagementAddOn("config-policy-controller")

			o := Options{
				Namespace:  "open-cluster-management-agent-addon",
				ConfigFile: "", // No config file
				Streams:    streams,
			}

			// Set up factory for the options
			o.ClusteradmFlags = testFlags

			clusters := []string{cluster1Name}

			err := o.runWithClient(clusterClient, addonClient, []string{"config-policy-controller"}, clusters)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			// Verify ManagedClusterAddOn was created without configs
			mca, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(cluster1Name).Get(
				context.Background(), "config-policy-controller", metav1.GetOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(mca.Spec.Configs).To(gomega.BeNil())
		})

		ginkgo.It("Should return error when config file does not exist", func() {
			assertCreatingClusters(cluster1Name)
			assertCreatingClusterManagementAddOn("argocd")

			o := Options{
				Namespace:  "open-cluster-management-agent-addon",
				ConfigFile: "/nonexistent/path/to/config.yaml",
				Streams:    streams,
			}

			// Set up factory for the options
			o.ClusteradmFlags = testFlags

			clusters := []string{cluster1Name}

			err := o.runWithClient(clusterClient, addonClient, []string{"argocd"}, clusters)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("failed to read config file"))
		})
	})

})
