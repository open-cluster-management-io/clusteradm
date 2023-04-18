// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
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
var clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

func TestIntegrationInstallAddons(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Integration install hub-addon Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	ginkgo.By("bootstrapping test environment")

	// start a kube-apiserver
	testEnv = &envtest.Environment{}

	cfg, err := testEnv.Start()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(cfg).ToNot(gomega.BeNil())

	kubeClient, err = kubernetes.NewForConfig(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	restConfig = cfg

	// add clusteradm flags
	f := cmdutil.NewFactory(TestClientGetter{cfg: cfg})
	clusteradmFlags = genericclioptionsclusteradm.NewClusteradmFlags(f)
})

var _ = ginkgo.AfterSuite(func() {
	ginkgo.By("tearing down the test environment")

	err := testEnv.Stop()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
})

type TestClientGetter struct {
	cfg *rest.Config
}

func (t TestClientGetter) ToRESTConfig() (*rest.Config, error) {
	return t.cfg, nil
}

func (t TestClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	discoveryClient, _ := discovery.NewDiscoveryClientForConfig(t.cfg)
	return memory.NewMemCacheClient(discoveryClient), nil
}

// ToRESTMapper returns a restmapper
func (t TestClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	client, _ := t.ToDiscoveryClient()
	return restmapper.NewDeferredDiscoveryRESTMapper(client), nil
}

// ToRawKubeConfigLoader return kubeconfig loader as-is
func (t TestClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return clientcmd.NewDefaultClientConfig(clientcmdapi.Config{}, nil)
}
