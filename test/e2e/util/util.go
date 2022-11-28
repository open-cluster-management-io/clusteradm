// Copyright Contributors to the Open Cluster Management project
package util

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"open-cluster-management.io/clusteradm/pkg/config"
)

// PrepareE2eEnvironment will init the e2e environment and join managedcluster1 to hub.
func PrepareE2eEnvironment() (*TestE2eConfig, error) {
	conf, err := initE2E()
	if err != nil {
		return nil, err
	}

	if err := conf.ResetEnv(); err != nil {
		return nil, err
	}

	return conf, nil
}

// initE2E get environment variables and init e2e environment.
func initE2E() (*TestE2eConfig, error) {

	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
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
		if err != nil {
			return nil, err
		}
		os.Setenv("KUBECONFIG", filepath.Join(home, ".kube", "config"))
	}

	e2eConf := NewTestE2eConfig(
		os.Getenv("KUBECONFIG"),
		os.Getenv("HUB_NAME"), os.Getenv("HUB_CTX"),
		os.Getenv("MANAGED_CLUSTER1_NAME"), os.Getenv("MANAGED_CLUSTER1_CTX"),
		os.Getenv("MANAGED_CLUSTER2_NAME"), os.Getenv("MANAGED_CLUSTER2_CTX"),
	)

	// clearenv set the e2e environment from initial state to empty
	clearenv := func() error {
		fmt.Println("cleaning hub...")
		fmt.Println("unjoin managedcluster1...")
		err := e2eConf.Clusteradm().Unjoin(
			"--context", e2eConf.Cluster().ManagedCluster1().Context(),
			"--cluster-name", e2eConf.Cluster().ManagedCluster1().Name(),
			"--purge-operator=false",
		)
		if err != nil {
			return err
		}
		err = WaitNamespaceDeleted(e2eConf.Kubeconfigpath, e2eConf.Cluster().ManagedCluster1().Context(), config.ManagedClusterNamespace)
		if err != nil {
			return err
		}

		fmt.Println("unjoin managedcluster2...")
		err = e2eConf.Clusteradm().Unjoin(
			"--context", e2eConf.Cluster().ManagedCluster2().Context(),
			"--cluster-name", e2eConf.Cluster().ManagedCluster2().Name(),
			"--purge-operator=false",
		)
		if err != nil {
			return err
		}
		err = WaitNamespaceDeleted(e2eConf.Kubeconfigpath, e2eConf.Cluster().ManagedCluster2().Context(), config.ManagedClusterNamespace)
		if err != nil {
			return err
		}

		// delete cluster finalizers
		err = DeleteClusterFinalizers(e2eConf.Kubeconfigpath, e2eConf.Cluster().Hub().Context())
		if err != nil {
			return err
		}
		err = e2eConf.Clusteradm().Clean(
			"--context", e2eConf.Cluster().Hub().Context(),
			"--purge-operator=false",
		)
		if err != nil {
			return err
		}

		err = DeleteClusterCSRs(e2eConf.Kubeconfigpath, e2eConf.Cluster().Hub().Context())
		if err != nil {
			return err
		}
		err = WaitNamespaceDeleted(e2eConf.Kubeconfigpath, e2eConf.Cluster().Hub().Context(), config.HubClusterNamespace)
		if err != nil {
			return err
		}

		// clear token and apiserver value
		e2eConf.clusteradm.h = HandledOutput{}
		return nil
	}
	// ClearEnv will unjoin managed cluster1 and clean hub.
	e2eConf.ClearEnv = clearenv

	return e2eConf, nil
}

func (tec *TestE2eConfig) ResetEnv() error {
	// ensure hub is initialized, and token and apiserver is set.
	fmt.Println("ensure hub is initialized...")
	err := tec.Clusteradm().Init(
		"--context", tec.Cluster().Hub().Context(),
		"--use-bootstrap-token",
		"--bundle-version=latest",
		"--wait",
	)
	if err != nil {
		return err
	}

	if tec.CommandResult() == nil || len(tec.CommandResult().RawCommand()) == 0 {
		err = tec.Clusteradm().Get("token")
		if err != nil {
			return err
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
		return err
	}

	err = tec.Clusteradm().Accept(
		"--clusters", tec.Cluster().ManagedCluster1().Name(),
		"--wait",
		"--context", tec.Cluster().Hub().Context(),
	)
	if err != nil {
		return err
	}

	// ensure managed cluster2 is not join-accepted
	fmt.Println("ensure managed cluster2 is unjoined...")
	err = tec.Clusteradm().Unjoin(
		"--context", tec.Cluster().ManagedCluster2().Context(),
		"--cluster-name", tec.Cluster().ManagedCluster2().Name(),
	)
	if err != nil {
		return err
	}

	fmt.Println("reset e2e environment finished.")
	return nil
}
