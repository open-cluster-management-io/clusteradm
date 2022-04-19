// Copyright Contributors to the Open Cluster Management project
package util

import (
	"fmt"
	. "github.com/onsi/gomega"
	"open-cluster-management.io/clusteradm/pkg/config"
	"os"
	"path"
	"path/filepath"
)

// PrepareE2eEnvironment will init the e2e environment and join managedcluster1 to hub.
func PrepareE2eEnvironment() *TestE2eConfig {
	conf := initE2E()

	conf.ResetEnv()

	return conf
}

// initE2E get environment variables and init e2e environment.
func initE2E() *TestE2eConfig {

	pwd, err := os.Getwd()
	Expect(err).To(BeNil())
	projectName := path.Base(path.Clean(path.Join(pwd, "..", "..", "..")))
	if v := os.Getenv("HUB_NAME"); v == "" {
		os.Setenv("HUB_NAME", projectName+"-e2e-test-hub")
	}
	if v := os.Getenv("HUB_CTX"); v == "" {
		os.Setenv("HUB_CTX", "kind-"+os.Getenv("HUB_NAME"))
	}
	if v := os.Getenv("MANAGED_CLUSTER1_NAME"); v == "" {
		os.Setenv("MANAGED_CLUSTER1_NAME", projectName+"-e2e-test-c1")
	}
	if v := os.Getenv("MANAGED_CLUSTER1_CTX"); v == "" {
		os.Setenv("MANAGED_CLUSTER1_CTX", "kind-"+os.Getenv("MANAGED_CLUSTER1_NAME"))
	}
	if v := os.Getenv("MANAGED_CLUSTER2_NAME"); v == "" {
		os.Setenv("MANAGED_CLUSTER2_NAME", projectName+"-e2e-test-c2")
	}
	if v := os.Getenv("MANAGED_CLUSTER2_CTX"); v == "" {
		os.Setenv("MANAGED_CLUSTER2_CTX", "kind-"+os.Getenv("MANAGED_CLUSTER2_NAME"))
	}
	if v := os.Getenv("KUBECONFIG"); v == "" {
		home, err := os.UserHomeDir()
		Expect(err).To(BeNil())
		os.Setenv("KUBECONFIG", filepath.Join(home, ".kube", "config"))
	}

	e2eConf := NewTestE2eConfig(
		os.Getenv("KUBECONFIG"),
		os.Getenv("HUB_NAME"), os.Getenv("HUB_CTX"),
		os.Getenv("MANAGED_CLUSTER1_NAME"), os.Getenv("MANAGED_CLUSTER1_CTX"),
		os.Getenv("MANAGED_CLUSTER2_NAME"), os.Getenv("MANAGED_CLUSTER2_CTX"),
	)

	// clearenv set the e2e environment from initial state to empty
	clearenv := func() {

		fmt.Println("unjoin managedcluster1...")
		err := e2eConf.Clusteradm().Unjoin(
			"--context", e2eConf.Cluster().ManagedCluster1().Context(),
			"--cluster-name", e2eConf.Cluster().ManagedCluster1().Name(),
		)
		if err != nil {
			panic(fmt.Sprintf("error occurred while unjoining managedcluster1: %s", err.Error()))
		}

		err = WaitNamespaceDeleted(e2eConf.Kubeconfigpath, e2eConf.Cluster().ManagedCluster1().Context(), config.ManagedClusterNamespace)
		if err != nil {
			panic(fmt.Sprintf("error occurred while unjoining managedcluster1: %s", err.Error()))
		}

		fmt.Println("cleaning hub...")
		err = e2eConf.Clusteradm().Clean("--context", e2eConf.Cluster().Hub().Context())
		if err != nil {
			panic(fmt.Sprintf("error occurred while cleaning hub: %s", err.Error()))
		}
		err = WaitNamespaceDeleted(e2eConf.Kubeconfigpath, e2eConf.Cluster().Hub().Context(), config.HubClusterNamespace)
		if err != nil {
			panic(fmt.Sprintf("error occurred while cleaning hub: %s", err.Error()))
		}
		err = WaitNamespaceDeleted(e2eConf.Kubeconfigpath, e2eConf.Cluster().Hub().Context(), config.OpenClusterManagementNamespace)
		if err != nil {
			panic(fmt.Sprintf("error occurred while cleaning hub: %s", err.Error()))
		}

		// clear token and apiserver value
		e2eConf.clusteradm.h = HandledOutput{}
	}
	// ClearEnv will unjoin managed cluster1 and clean hub.
	e2eConf.ClearEnv = clearenv

	return e2eConf
}

func (tec *TestE2eConfig) ResetEnv() {
	// ensure hub is initialized, and token and apiserver is set.
	fmt.Println("ensure hub is initialized...")
	err := tec.Clusteradm().Init(
		"--context", tec.Cluster().Hub().Context(),
		"--use-bootstrap-token",
		"--wait",
	)
	if err != nil {
		panic(fmt.Sprintf("error occurred while initializing hub: %s", err))
	}

	if tec.CommandResult() == nil || len(tec.CommandResult().RawCommand()) == 0 {
		err = tec.Clusteradm().Get("token")
		if err != nil {
			panic(fmt.Sprintf("error occurred while get token from hub: %s", err))
		}
	}

	// ensure managed cluster1 is join-accepted
	fmt.Println("ensure managed cluster1 is join and accepted...")
	err = tec.Clusteradm().Join(
		"--context", tec.Cluster().ManagedCluster1().Context(),
		"--hub-token", tec.CommandResult().Token(), "--hub-apiserver", tec.CommandResult().Host(),
		"--cluster-name", tec.Cluster().ManagedCluster1().Name(),
		"--wait",
		"--force-internal-endpoint-lookup",
	)
	if err != nil {
		panic(fmt.Sprintf("error occurred while managedcluster1 joining hub: %s", err))
	}

	err = tec.Clusteradm().Accept(
		"--clusters", tec.Cluster().ManagedCluster1().Name(),
		"--wait", "30",
		"--context", tec.Cluster().Hub().Context(),
	)
	if err != nil {
		panic(fmt.Sprintf("error occurred while hub accepting managedcluster1: %s", err))
	}

	// ensure managed cluster2 is not join-accepted
	fmt.Println("ensure managed cluster2 is unjoined...")
	err = tec.Clusteradm().Unjoin(
		"--context", tec.Cluster().ManagedCluster2().Context(),
		"--cluster-name", tec.Cluster().ManagedCluster2().Name(),
	)
	if err != nil {
		// TODO: figure out how to catch this error and then use panic here(when unjoin a unjoined managedcluster, this error occurred)
		// 2022/03/04 06:43:00 the server could not find the requested resource (get appliedmanifestworks.work.open-cluster-management.io)
		fmt.Printf("error occurred while unjoining managedcluster2: %s\n", err.Error())
	}

	fmt.Println("reset e2e environment finished.")
}
