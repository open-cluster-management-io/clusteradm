// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"os"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	addonclientset "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"open-cluster-management.io/clusteradm/test/e2e/util"
)

var e2e *util.TestE2eConfig

// var testEnv *envtest.Environment
var restConfig *rest.Config
var kubeClient, managedClusterKubeClient kubernetes.Interface
var apiExtensionsClient apiextensionsclient.Interface
var dynamicClient dynamic.Interface
var clusterClient clusterv1client.Interface
var addonClient addonclientset.Interface
var operatorClient operatorclient.Interface

func TestE2EClusteradm(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	// fetch the current config
	suiteConfig, reporterConfig := ginkgo.GinkgoConfiguration()
	// adjust it
	suiteConfig.SkipStrings = []string{"NEVER-RUN"}
	// suiteConfig.FocusStrings = []string{"test clusteradm with manual bootstrap token"}
	reporterConfig.FullTrace = true

	ginkgo.RunSpecs(t, "E2E clusteradm test", suiteConfig, reporterConfig)
}

var _ = ginkgo.BeforeSuite(func() {
	ginkgo.By("Starting e2e test environment")

	var err error

	// set cluster info and start clusters.
	e2e, err = util.PrepareE2eEnvironment()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	pathOptions := clientcmd.NewDefaultPathOptions()
	configapi, err := pathOptions.GetStartingConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	configapi.CurrentContext = os.Getenv("HUB_CTX")
	clientConfig := clientcmd.NewDefaultClientConfig(*configapi, nil)

	hubConfig, err := clientConfig.ClientConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	kubeClient, err = kubernetes.NewForConfig(hubConfig)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	managedClusterCtx := os.Getenv("MANAGED_CLUSTER1_CTX")
	managedClusterClientConfig := clientcmd.NewNonInteractiveClientConfig(*configapi,
		managedClusterCtx, nil, clientcmd.NewDefaultClientConfigLoadingRules())
	managedClusterConfig, err := managedClusterClientConfig.ClientConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	managedClusterKubeClient, err = kubernetes.NewForConfig(managedClusterConfig)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	apiExtensionsClient, err = apiextensionsclient.NewForConfig(hubConfig)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	dynamicClient, err = dynamic.NewForConfig(hubConfig)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	clusterClient, err = clusterv1client.NewForConfig(hubConfig)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	addonClient, err = addonclientset.NewForConfig(hubConfig)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	operatorClient, err = operatorclient.NewForConfig(hubConfig)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	err = clusterv1.Install(scheme.Scheme)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	restConfig = hubConfig
})
