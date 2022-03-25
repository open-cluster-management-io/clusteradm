// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	eventuallyTimeout    = 30 // seconds
	eventuallyInterval   = 1  // seconds
	consistentlyTimeout  = 3  // seconds
	consistentlyInterval = 1  // seconds
)

var testEnv *envtest.Environment
var restConfig *rest.Config
var kubeClient kubernetes.Interface
var apiExtensionsClient apiextensionsclient.Interface
var dynamicClient dynamic.Interface

func TestIntegrationInstallAddons(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Integration install hub-addon Suite")
}

var _ = ginkgo.BeforeSuite(func(done ginkgo.Done) {
	ginkgo.By("bootstrapping test environment")

	// start a kube-apiserver
	testEnv = &envtest.Environment{}

	cfg, err := testEnv.Start()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(cfg).ToNot(gomega.BeNil())

	kubeClient, err = kubernetes.NewForConfig(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	apiExtensionsClient, err = apiextensionsclient.NewForConfig(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	dynamicClient, err = dynamic.NewForConfig(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	restConfig = cfg
	close(done)
}, 60)

var _ = ginkgo.AfterSuite(func() {
	ginkgo.By("tearing down the test environment")

	err := testEnv.Stop()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
})
