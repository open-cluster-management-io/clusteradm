// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"context"
	"os"
	"time"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"open-cluster-management.io/clusteradm/pkg/config"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"open-cluster-management.io/clusteradm/test/e2e/clusteradm/scenario"
)

var _ = ginkgo.Describe("test clusteradm with manual bootstrap token", ginkgo.Label("join-hub-skip-approve"), func() {
	ginkgo.BeforeEach(func() {
		err := e2e.ClearEnv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("join hub scenario with manual bootstrap token", func() {
		var err error
		ginkgo.It("should managedclusters join and accepted successfully", func() {
			ginkgo.By("init hub with manual bootstrap token")
			clusterAdm := e2e.Clusteradm()
			err = clusterAdm.Init(
				"--timeout", "400",
				"--context", e2e.Cluster().Hub().Context(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm init error")
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
				KubeConfig: ptr.To[string](e2e.KubeConfigPath),
				Context:    ptr.To[string](e2e.Cluster().Hub().Context()),
			}
			r := reader.NewResourceReader(cmdutil.NewFactory(kubeConfigFlags), false, genericiooptions.IOStreams{Out: os.Stdout, ErrOut: os.Stderr})
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
							ExpirationSeconds: ptr.To[int64](3600),
						},
					}, metav1.CreateOptions{})
				if err != nil {
					return err
				}
				token = tr.Status.Token
				return nil
			}, 30, 1).Should(gomega.BeNil())

			ginkgo.By("managedcluster1 join hub")
			gomega.Eventually(func() error {
				return e2e.Clusteradm().Join(
					"--context", e2e.Cluster().ManagedCluster1().Context(),
					"--hub-token", token,
					"--hub-apiserver", clusterAdm.Result().Host(),
					"--force-internal-endpoint-lookup",
					"--cluster-name", e2e.Cluster().ManagedCluster1().Name(),
					"--wait",
				)
			}, 2*time.Minute, 10*time.Second).Should(gomega.BeNil(), "managedcluster1 join error")

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
