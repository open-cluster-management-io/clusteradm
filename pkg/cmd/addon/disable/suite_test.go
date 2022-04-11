// Copyright Contributors to the Open Cluster Management project
package disable

import (
	"path/filepath"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
)

var testEnv *envtest.Environment
var restConfig *rest.Config
var kubeClient kubernetes.Interface
var apiExtensionsClient apiextensionsclient.Interface
var dynamicClient dynamic.Interface
var clusterClient clusterv1client.Interface
var addonClient addonv1alpha1client.Interface

func TestIntegrationEnableAddons(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Integration Disable Addons Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	ginkgo.By("bootstrapping test environment")

	// start a kube-apiserver
	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "..", "vendor", "open-cluster-management.io", "api", "cluster", "v1"),
			filepath.Join("..", "..", "..", "..", "vendor", "open-cluster-management.io", "api", "addon", "v1alpha1"),
		},
	}

	cfg, err := testEnv.Start()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(cfg).ToNot(gomega.BeNil())

	kubeClient, err = kubernetes.NewForConfig(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	apiExtensionsClient, err = apiextensionsclient.NewForConfig(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	dynamicClient, err = dynamic.NewForConfig(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	clusterClient, err = clusterv1client.NewForConfig(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	addonClient, err = addonv1alpha1client.NewForConfig(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	restConfig = cfg
})

var _ = ginkgo.AfterSuite(func() {
	ginkgo.By("tearing down the test environment")

	err := testEnv.Stop()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
})
