// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"open-cluster-management.io/clusteradm/pkg/helpers/version"
)

var _ = ginkgo.Describe("test clusteradm upgrade clustermanager and Klusterlets", ginkgo.Ordered, func() {

	ginkgo.BeforeAll(func() {
		ginkgo.By("reset e2e environment...")
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	var err error

	ginkgo.It("run cluster manager upgrade version latest ", func() {
		ginkgo.By("init hub with service account")
		err = e2e.Clusteradm().Init(
			"--context", e2e.Cluster().Hub().Context(),
		)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")

		ginkgo.By("Check the version of operator and agent")
		gomega.Eventually(func() error {
			operator, err := kubeClient.AppsV1().Deployments("open-cluster-management").Get(context.TODO(), "cluster-manager", metav1.GetOptions{})
			if err != nil {
				return err
			}
			if operator.Spec.Template.Spec.Containers[0].Image != "quay.io/open-cluster-management/registration-operator:v"+version.GetDefaultBundleVersion() {
				return fmt.Errorf("version of the operator is not correct, get %s", operator.Spec.Template.Spec.Containers[0].Image)
			}
			registration, err := kubeClient.AppsV1().Deployments("open-cluster-management-hub").Get(
				context.TODO(), "cluster-manager-registration-controller", metav1.GetOptions{})
			if err != nil {
				return err
			}
			if registration.Spec.Template.Spec.Containers[0].Image != "quay.io/open-cluster-management/registration:v"+version.GetDefaultBundleVersion() {
				return fmt.Errorf("version of the registration agent is not correct, get %s", operator.Spec.Template.Spec.Containers[0].Image)
			}
			return nil
		}, 120*time.Second, 5*time.Second).Should(gomega.Succeed())

		err = e2e.Clusteradm().Upgrade(
			"clustermanager",
			"--bundle-version=latest",
			"--context", e2e.Cluster().Hub().Context(),
		)

		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm upgrade error")

		ginkgo.By("Upgrade to the latest version")
		gomega.Eventually(func() error {
			operator, err := kubeClient.AppsV1().Deployments("open-cluster-management").Get(context.TODO(), "cluster-manager", metav1.GetOptions{})
			if err != nil {
				return err
			}
			if operator.Spec.Template.Spec.Containers[0].Image != "quay.io/open-cluster-management/registration-operator:latest" {
				return fmt.Errorf("version of the operator is not correct, get %s", operator.Spec.Template.Spec.Containers[0].Image)
			}
			registration, err := kubeClient.AppsV1().Deployments("open-cluster-management-hub").Get(
				context.TODO(), "cluster-manager-registration-controller", metav1.GetOptions{})
			if err != nil {
				return err
			}
			if registration.Spec.Template.Spec.Containers[0].Image != "quay.io/open-cluster-management/registration:latest" {
				return fmt.Errorf("version of the operator is not correct, get %s", operator.Spec.Template.Spec.Containers[0].Image)
			}
			return nil
		}, 120*time.Second, 5*time.Second).Should(gomega.Succeed())
	})
})
