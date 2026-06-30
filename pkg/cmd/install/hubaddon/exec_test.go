// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"context"
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"open-cluster-management.io/clusteradm/pkg/helpers/helm"
)

var _ = ginkgo.Describe("install hub-addon", func() {
	ginkgo.Context("validate", func() {
		ginkgo.It("Should not create any built-in add-on deployment(s) because it's not a valid add-on name", func() {
			o := Options{
				ClusteradmFlags: clusteradmFlags,
				names:           "no-such-addon",
			}

			err := o.validate()
			gomega.Expect(err).To(gomega.HaveOccurred())
		})

		ginkgo.It("Should not create any built-in add-on deployment(s) because it's not a valid version", func() {
			o := Options{
				ClusteradmFlags: clusteradmFlags,
				chartVersion:    "invalid",
			}

			err := o.validate()
			gomega.Expect(err).Should(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("install policy addon", func() {
		ginkgo.It("Should deploy the policy add-on deployments in open-cluster-management namespace successfully", func() {
			o := Options{
				ClusteradmFlags: clusteradmFlags,
				namespace:       ocmNamespace,
				Streams:         genericiooptions.IOStreams{Out: os.Stdout, ErrOut: os.Stderr},
				Helm:            helm.NewHelm(),
			}

			err := o.runWithHelmClient("governance-policy-framework")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			var policyAddonDeployments = []string{
				"governance-policy-propagator",
				"governance-policy-addon-controller",
			}

			for _, deployment := range policyAddonDeployments {
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
	})
})
