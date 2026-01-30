// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"io"
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"open-cluster-management.io/clusteradm/pkg/cmd/install/hubaddon/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers/helm"
	"open-cluster-management.io/clusteradm/pkg/version"
)

var _ = ginkgo.Describe("install hub-addon", func() {
	var policyAddonDeployments = []string{
		"governance-policy-propagator",
		"governance-policy-addon-controller",
	}

	const (
		invalidAddon = "no-such-addon"
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
		ocmBundleVersion, err = version.GetVersionBundle(ocmVersion, "")
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

	ginkgo.Context("installPolicyAddon", func() {
		ginkgo.It("Should deploy the policy add-on deployments in open-cluster-management namespace successfully", func(ctx ginkgo.SpecContext) {
			o := Options{
				ClusteradmFlags: clusteradmFlags,
				bundleVersion:   ocmVersion,
				values: scenario.Values{
					Namespace:     ocmNamespace,
					BundleVersion: ocmBundleVersion,
				},
				Streams: genericiooptions.IOStreams{Out: os.Stdout, ErrOut: os.Stderr},
			}

			err := o.installPolicyAddon(ctx)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			for _, deployment := range policyAddonDeployments {
				gomega.Eventually(func() (bool, error) {
					appDeployment, err := kubeClient.AppsV1().Deployments(ocmNamespace).Get(ctx, deployment, metav1.GetOptions{})
					if err != nil {
						return false, err
					}

					availableReplicas := appDeployment.Status.AvailableReplicas
					expectedReplicas := appDeployment.Status.Replicas
					return availableReplicas == expectedReplicas, nil
				}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue(), deployment+" deployment should be ready")
			}
		})
	})

	ginkgo.Context("runWithHelmClient", func() {
		ginkgo.It("Should not create any built-in add-on deployment(s) in dry-run mode", func(ctx ginkgo.SpecContext) {
			addon := ArgocdAddonName
			clusteradmFlagsCopy := *clusteradmFlags
			clusteradmFlagsCopy.DryRun = true
			streams := genericiooptions.IOStreams{Out: io.Discard, ErrOut: os.Stderr}
			o := Options{
				ClusteradmFlags: &clusteradmFlagsCopy,
				values: scenario.Values{
					CreateNamespace: true,
				},
				Streams: streams,
				Helm:    helm.NewHelm(&clusteradmFlagsCopy, streams),
			}

			err := o.runWithHelmClient(ctx, addon)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Consistently(func(g gomega.Gomega) bool {
				_, err := kubeClient.CoreV1().Namespaces().Get(ctx, addon, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue(), addon+" namespace should not be created")
		})

		ginkgo.It("Should install the argocd-agent add-on in dry-run mode", func(ctx ginkgo.SpecContext) {
			addon := ArgocdAgentAddonName
			addonChart := AddonCharts[addon]
			clusteradmFlagsCopy := *clusteradmFlags
			clusteradmFlagsCopy.DryRun = true
			streams := genericiooptions.IOStreams{Out: io.Discard, ErrOut: os.Stderr}
			o := Options{
				ClusteradmFlags: &clusteradmFlagsCopy,
				values: scenario.Values{
					CreateNamespace: true,
				},
				Streams: streams,
				Helm:    helm.NewHelm(&clusteradmFlagsCopy, streams),
			}

			err := o.runWithHelmClient(ctx, addon)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Consistently(func(g gomega.Gomega) bool {
				_, err := kubeClient.CoreV1().Namespaces().Get(ctx, addonChart.Namespace, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue(), addonChart.Namespace+" namespace should not be created")
		})
	})
})
