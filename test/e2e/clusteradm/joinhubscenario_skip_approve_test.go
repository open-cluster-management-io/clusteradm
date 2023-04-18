// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"context"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"os"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/pointer"
	"open-cluster-management.io/clusteradm/pkg/config"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"open-cluster-management.io/clusteradm/test/e2e/clusteradm/scenario"
)

var _ = ginkgo.Describe("test clusteradm with manual bootstrap token", func() {
	ginkgo.BeforeEach(func() {
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("join hub scenario with manual bootstrap token", func() {
		var err error
		ginkgo.It("should managedclusters join and accepted successfully", func() {
			ginkgo.By("init hub with manual bootstrap token")
			err = e2e.Clusteradm().Init(
				"--timeout", "400",
				"--context", e2e.Cluster().Hub().Context(),
				"--bundle-version=latest",
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

			kubeConfigFlags := &genericclioptions.ConfigFlags{
				KubeConfig: pointer.String(e2e.Kubeconfigpath),
				Context:    pointer.String(e2e.Cluster().Hub().Context()),
			}
			resourceBuilder := cmdutil.NewFactory(kubeConfigFlags).NewBuilder()
			r := reader.NewResourceReader(resourceBuilder, false, genericclioptions.IOStreams{Out: os.Stdout, ErrOut: os.Stderr})
			var values = make(map[string]interface{})
			err = r.Apply(scenario.Files, values, files...)
			gomega.Expect(err).To(gomega.BeNil())

			var token string
			gomega.Eventually(func() error {
				tr, err := kubeClient.CoreV1().
					ServiceAccounts(config.OpenClusterManagementNamespace).
					CreateToken(context.TODO(), "sa-manual-token", &authv1.TokenRequest{
						Spec: authv1.TokenRequestSpec{
							// token expired in 1 hour
							ExpirationSeconds: pointer.Int64(3600),
						},
					}, metav1.CreateOptions{})
				if err != nil {
					return err
				}
				token = tr.Status.Token
				return nil
			}, 30, 1).Should(gomega.BeNil())

			ginkgo.By("managedcluster1 join hub")
			err = e2e.Clusteradm().Join(
				"--context", e2e.Cluster().ManagedCluster1().Context(),
				"--hub-token", token,
				"--hub-apiserver", e2e.CommandResult().Host(),
				"--force-internal-endpoint-lookup",
				"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
				"--bundle-version=latest",
				"--wait",
			)

			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "managedcluster1 join error")
			ginkgo.By("hub accept managedcluster1")
			err = e2e.Clusteradm().Accept(
				"--clusters", e2e.Cluster().ManagedCluster1().Name(),
				"--wait",
				"--context", e2e.Cluster().Hub().Context(),
				"--skip-approve-check",
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm accept error")
		})
	})
})
