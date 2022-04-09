// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"open-cluster-management.io/clusteradm/pkg/config"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"
	scenario "open-cluster-management.io/clusteradm/test/e2e/clusteradm/scenario"
)

var _ = ginkgo.Describe("test clusteradm with manual bootstrap token", func() {
	ginkgo.BeforeEach(func() {
		e2e.ClearEnv()
	})

	ginkgo.Context("join hub scenario with manual bootstrap token", func() {
		var err error
		ginkgo.It("should managedclusters join and accepted successfully", func() {
			ginkgo.By("init hub with manual bootstrap token")
			err = e2e.Clusteradm().Init(
				"--timeout", "400",
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")
			time.Sleep(1000000)
			ginkgo.By("init files with manual bootstrap token")
			files := []string{
				"init/namespace.yaml",
			}
			files = append(files,
				"init/bootstrap_sa_manual_token.yaml",
				"init/bootstrap_cluster_role.yaml",
				"init/bootstrap_sa_cluster_role_binding_manual_token.yaml",
			)
			reader := scenario.GetScenarioResourcesReader()
			applierBuilder := &apply.ApplierBuilder{}
			var values = make(map[string]interface{})
			applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()
			_, err = applier.ApplyDirectly(reader, values, false, "", files...)
			gomega.Expect(err).To(gomega.BeNil())
			sa, err := kubeClient.CoreV1().
				ServiceAccounts(config.OpenClusterManagementNamespace).
				Get(context.TODO(), "sa-manual-token", metav1.GetOptions{})
			gomega.Expect(err).To(gomega.BeNil())
			var foundSecret *corev1.Secret
			prefix := "sa-manual-token"
			if len(prefix) > 63 {
				prefix = prefix[:37]
			}
			gomega.Eventually(func() error {
				secrets, err := kubeClient.CoreV1().
					Secrets(config.OpenClusterManagementNamespace).
					List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					return err
				}
				for _, secret := range secrets.Items {
					if strings.HasPrefix(secret.Name, prefix) {
						foundSecret = &secret
						break
					}
				}
				if foundSecret == nil {
					return fmt.Errorf("Secret with prefix %s not found, trying again", prefix)
				}
				return nil
			}, 30, 1).Should(gomega.BeNil())

			gomega.Expect(sa.Name).To(gomega.Equal("sa-manual-token"))
			token := string(foundSecret.Data["token"])
			fmt.Println("find token", token)

			ginkgo.By("managedcluster1 join hub")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", token,
				"--hub-apiserver", e2e.CommandResult().Host(),
				"--force-internal-endpoint-lookup",
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
			)

			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedcluster1 join error")
			ginkgo.By("hub accept managedcluster1")
			err = e2e.Clusteradm().Accept(
				"--clusters", e2e.Cluster().ManagedCluster1().Name(),
				"--wait", "30",
				"--context", e2e.Cluster().Hub().Context(),
				"--skip-approve-check",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm accept error")
		})
	})
})
