// Copyright Contributors to the Open Cluster Management project
package enable

import (
	"path/filepath"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// nolint:deadcode,varcheck
const (
	eventuallyTimeout    = 30 // seconds
	eventuallyInterval   = 1  // seconds
	consistentlyTimeout  = 3  // seconds
	consistentlyInterval = 1  // seconds
)

var testEnv *envtest.Environment
var restConfig *rest.Config
var kubeClient kubernetes.Interface
var clusterClient clusterv1client.Interface
var addonClient addonv1alpha1client.Interface
var testFlags *genericclioptionsclusteradm.ClusteradmFlags

func TestIntegrationEnableAddons(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Integration Enable Addons Suite")
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
	clusterClient, err = clusterv1client.NewForConfig(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	addonClient, err = addonv1alpha1client.NewForConfig(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	restConfig = cfg

	// Set up testFlags with a factory that uses the test config
	f := cmdutil.NewFactory(TestClientGetter{cfg: cfg})
	testFlags = genericclioptionsclusteradm.NewClusteradmFlags(f)
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
