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
	"open-cluster-management.io/clusteradm/pkg/helpers/helm"
)

var _ = ginkgo.Describe("install hub-addon", func() {
	ginkgo.Context("validate", func() {
		ginkgo.It("Should not create any built-in add-on deployment(s) because it's not a valid version", func() {
			o := Options{
				ClusteradmFlags: clusteradmFlags,
				bundleVersion:   "invalid",
			}

			err := o.validate()
			gomega.Expect(err).Should(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("install policy addon", func() {
		ginkgo.It("Should deploy the policy add-on deployments in open-cluster-management namespace successfully", func(ctx ginkgo.SpecContext) {
			streams := genericiooptions.IOStreams{Out: os.Stdout, ErrOut: os.Stderr}
			o := Options{
				ClusteradmFlags: clusteradmFlags,
				Streams:         streams,
				Helm:            helm.NewHelm(clusteradmFlags, streams),
			}

			err := o.runWithHelmClient(PolicyFrameworkAddonName)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			var policyAddonDeployments = []string{
				"governance-policy-propagator",
				"governance-policy-addon-controller",
			}

			for _, deployment := range policyAddonDeployments {
				gomega.Eventually(ctx, func() (bool, error) {
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

	ginkgo.It("Should not create any built-in add-on deployment(s) in dry-run mode", func(ctx ginkgo.SpecContext) {
		addon := "argocd"
		clusteradmFlagsCopy := *clusteradmFlags
		clusteradmFlagsCopy.DryRun = true
		streams := genericiooptions.IOStreams{Out: io.Discard, ErrOut: os.Stderr}
		o := Options{
			ClusteradmFlags: &clusteradmFlagsCopy,
			createNamespace: true,
			Streams:         streams,
			Helm:            helm.NewHelm(&clusteradmFlagsCopy, streams),
		}

		err := o.runWithHelmClient(addon)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		gomega.Consistently(func(g gomega.Gomega) bool {
			_, err := kubeClient.CoreV1().Namespaces().Get(ctx, addon, metav1.GetOptions{})
			return errors.IsNotFound(err)
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue(), addon+" namespace should not be created")
	})
})
