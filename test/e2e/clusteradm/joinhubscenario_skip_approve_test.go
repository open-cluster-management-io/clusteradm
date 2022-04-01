// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"context"
	"fmt"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"open-cluster-management.io/clusteradm/pkg/config"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"
	scenario "open-cluster-management.io/clusteradm/test/e2e/clusteradm/scenario"
	"strings"
	"time"
)
var _ = ginkgo.Describe("test clusteradm with manual bootstrap token", func() {
	ginkgo.BeforeEach(func() {
		e2e.ClearEnv()
	})

	ginkgo.AfterEach(func() {
		ginkgo.By("reset e2e environment...")
		e2e.ResetEnv()
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
			var secret *corev1.Secret
			var prefix string
			for _, objectRef := range sa.Secrets {
				if objectRef.Namespace != "" && objectRef.Namespace != config.OpenClusterManagementNamespace {
					continue
				}
				prefix = "sa-manual-token"
				if len(prefix) > 63 {
					prefix = prefix[:37]
				}
				if strings.HasPrefix(objectRef.Name, prefix) {
					secret, err = kubeClient.CoreV1().
						Secrets(config.OpenClusterManagementNamespace).
						Get(context.TODO(), objectRef.Name, metav1.GetOptions{})
					if err != nil {
						continue
					}
					if secret.Type == corev1.SecretTypeServiceAccountToken {
						break
					}
				}
			}
			gomega.Expect(sa.Name).To(gomega.Equal("sa-manual-token"))
			fmt.Println("service account", sa)
			fmt.Println("secret", secret)
			time.Sleep(1000000)
			//token := string(secret.Data["token"])

		//	fmt.Println("tokeeeeeeeeeeeeeeeeen", token)
			//fmt.Println("secretttttttttttttDattttttttta", secret.Data)

			ginkgo.By("managedcluster1 join hub")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", e2e.CommandResult().Token(),
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