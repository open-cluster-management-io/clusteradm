// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ocmNamespace           = "open-cluster-management"
	channelDeployment      = "multicluster-operators-channel"
	subscriptionDeployment = "multicluster-operators-subscription"
	propagatorDeployment   = "governance-policy-propagator"
	policyAddonDeployment  = "governance-policy-addon-controller"
)

var _ = ginkgo.Describe("install hub-addon", func() {

	ginkgo.Context("runWithClient", func() {

		ginkgo.It("Should not create any built-in add-on deployment(s) because it's not a valid add-on name", func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ocmNamespace,
				},
			}
			_, err := kubeClient.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			o := Options{
				values: Values{
					hubAddons: []string{"no-such-addon"},
				},
			}

			err = o.runWithClient(kubeClient, apiExtensionsClient, dynamicClient, false)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Consistently(func() error {
				_, err := kubeClient.AppsV1().Deployments(ocmNamespace).Get(context.Background(), channelDeployment, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, consistentlyTimeout, consistentlyInterval).Should(gomega.HaveOccurred())

			gomega.Consistently(func() error {
				_, err := kubeClient.AppsV1().Deployments(ocmNamespace).Get(context.Background(), subscriptionDeployment, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, consistentlyTimeout, consistentlyInterval).Should(gomega.HaveOccurred())
		})

		ginkgo.It("Should deploy the built in application-manager add-on deployments in open-cluster-management namespace successfully", func() {
			o := Options{
				values: Values{
					hubAddons: []string{appMgrAddonName},
				},
			}

			err := o.runWithClient(kubeClient, apiExtensionsClient, dynamicClient, false)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.AppsV1().Deployments(ocmNamespace).Get(context.Background(), channelDeployment, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.AppsV1().Deployments(ocmNamespace).Get(context.Background(), subscriptionDeployment, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("Should deploy the built-in policy-framework add-on deployments in open-cluster-management namespace successfully", func() {
			o := Options{
				values: Values{
					hubAddons: []string{policyFrameworkAddonName},
				},
			}

			err := o.runWithClient(kubeClient, apiExtensionsClient, dynamicClient, false)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.AppsV1().Deployments(ocmNamespace).Get(context.Background(), propagatorDeployment, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.AppsV1().Deployments(ocmNamespace).Get(context.Background(), policyAddonDeployment, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})
	})
})
