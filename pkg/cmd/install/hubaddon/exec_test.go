// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"context"
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"open-cluster-management.io/clusteradm/pkg/cmd/install/hubaddon/scenario"
	"open-cluster-management.io/clusteradm/pkg/version"
)

var _ = ginkgo.Describe("install hub-addon", func() {

	// Map of Hub Addons to test, mapping the Hub Addon name to
	// an array of deployment names to check for availability
	var hubAddons = map[string][]string{
		"governance-policy-framework": {
			"governance-policy-propagator",
			"governance-policy-addon-controller",
		},
	}

	const (
		invalidNamespace = "no-such-ns"
		invalidAddon     = "no-such-addon"
	)

	var (
		ocmVersion       = version.GetDefaultBundleVersion()
		ocmBundleVersion = version.VersionBundle{}
	)

	ginkgo.BeforeEach(func() {
		if bundleVersion, ok := os.LookupEnv("OCM_BUNDLE_VERSION"); ok && bundleVersion != "" {
			ocmVersion = bundleVersion
		}

		var err error
		ocmBundleVersion, err = version.GetVersionBundle(ocmVersion)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	ginkgo.Context("validate", func() {

		ginkgo.It("Should not create any built-in add-on deployment(s) because it's not a valid add-on name", func() {
			o := Options{
				ClusteradmFlags: clusteradmFlags,
				names:           invalidAddon,
			}

			err := o.validate()
			gomega.Expect(err).To(gomega.HaveOccurred())
		})

		ginkgo.It("Should not create any built-in add-on deployment(s) because it's not a valid version", func() {
			o := Options{
				ClusteradmFlags: clusteradmFlags,
				bundleVersion:   "invalid",
			}

			err := o.validate()
			gomega.Expect(err).Should(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("runWithClient - invalid configurations", func() {

		ginkgo.It("Should not create any built-in add-on deployment(s) because it's not a valid add-on name", func() {
			o := Options{
				ClusteradmFlags: clusteradmFlags,
				values: scenario.Values{
					HubAddons: []string{invalidAddon},
				},
			}

			err := o.runWithClient()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			for _, addonDeployments := range hubAddons {
				for _, deployment := range addonDeployments {
					gomega.Consistently(func() bool {
						_, err := kubeClient.AppsV1().Deployments(ocmNamespace).Get(context.Background(), deployment, metav1.GetOptions{})
						return errors.IsNotFound(err)
					}, consistentlyTimeout, consistentlyInterval).Should(gomega.BeTrue())
				}
			}
		})

		ginkgo.It("Should not create any built-in add-on deployment(s) because it's not a valid namespace", func() {
			o := Options{
				ClusteradmFlags: clusteradmFlags,
				bundleVersion:   ocmVersion,
				values: scenario.Values{
					Namespace: invalidNamespace,
					HubAddons: []string{scenario.PolicyFrameworkAddonName},
				},
				Streams: genericiooptions.IOStreams{Out: os.Stdout, ErrOut: os.Stderr},
			}

			err := o.runWithClient()
			gomega.Expect(err).Should(gomega.HaveOccurred())
		})
	})

	// Generate entries for the `runWithClient` test table
	addonTests := []ginkgo.TableEntry{}
	for hubAddon, deployments := range hubAddons {
		addonTests = append(addonTests, ginkgo.Entry(hubAddon, hubAddon, deployments))
	}

	ginkgo.DescribeTableSubtree("runWithClient",
		func(hubAddon string, deployments []string) {
			ginkgo.It("Should deploy the built in "+hubAddon+" add-on deployments in open-cluster-management namespace successfully", func() {
				o := Options{
					ClusteradmFlags: clusteradmFlags,
					bundleVersion:   ocmVersion,
					values: scenario.Values{
						Namespace:     ocmNamespace,
						HubAddons:     []string{hubAddon},
						BundleVersion: ocmBundleVersion,
					},
					Streams: genericiooptions.IOStreams{Out: os.Stdout, ErrOut: os.Stderr},
				}

				err := o.runWithClient()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				for _, deployment := range deployments {
					gomega.Eventually(func() (bool, error) {
						appDeployment, err := kubeClient.AppsV1().Deployments(ocmNamespace).Get(context.Background(), deployment, metav1.GetOptions{})
						if err != nil {
							return false, err
						}

						availableReplicas := appDeployment.Status.AvailableReplicas
						expectedReplicas := appDeployment.Status.Replicas
						return availableReplicas == expectedReplicas, nil
					}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue(), deployment+" deployment should be ready")
				}
			})
		},
		addonTests,
	)
})
