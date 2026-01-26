// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/scheme"
	addonclientset "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"open-cluster-management.io/clusteradm/test/e2e/util"
)

var e2e *util.TestE2eConfig

// var testEnv *envtest.Environment
var (
	kubeClient, managedClusterKubeClient kubernetes.Interface
	dynamicClient                        dynamic.Interface
	clusterClient                        clusterv1client.Interface
	addonClient                          addonclientset.Interface
	operatorClient                       operatorclient.Interface

	// bundleVersion is the current version of clusteradm used in e2e
	bundleVersion string
)

func init() {
	flag.StringVar(&bundleVersion, "bundle-version", "latest", "bundle version of the clusteradm")
}

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

	logger := zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true))
	logf.SetLogger(logger)
	var err error

	// set cluster info and start clusters.
	e2e, err = util.PrepareE2eEnvironment(bundleVersion)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	hubConfig := e2e.HubKubeConfig()
	kubeClient, err = kubernetes.NewForConfig(hubConfig)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	managedClusterConfig := e2e.ManagedClusterKubeConfig()
	managedClusterKubeClient, err = kubernetes.NewForConfig(managedClusterConfig)
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
})
