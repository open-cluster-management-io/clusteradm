// Copyright Contributors to the Open Cluster Management project
package util

import (
	"fmt"
	"open-cluster-management.io/clusteradm/pkg/config"
	"os"
	"path"
	"path/filepath"
)

// PrepareE2eEnvironment will init the e2e environment and join managedcluster1 to hub.
func PrepareE2eEnvironment(version string) (*TestE2eConfig, error) {
	conf, err := initE2E(version)
	if err != nil {
		return nil, err
	}

	if err := conf.ResetEnv(); err != nil {
		return nil, err
	}

	return conf, nil
}

// initE2E get environment variables and init e2e environment.
func initE2E(version string) (*TestE2eConfig, error) {

	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	projectName := path.Base(path.Clean(path.Join(pwd, "..", "..", "..")))

	var hubCtx, mcl1Ctx, kubeConfigPath, hubName, mcl1Name string

	if hubName = os.Getenv("HUB_NAME"); hubName == "" {
		hubName = projectName + "-e2e-test-hub"
	}
	if hubCtx = os.Getenv("HUB_CTX"); hubCtx == "" {
		hubCtx = "kind-" + hubName
	}
	if mcl1Name = os.Getenv("MANAGED_CLUSTER1_NAME"); mcl1Name == "" {
		mcl1Name = projectName + "-e2e-test-c1"
	}
	if mcl1Ctx = os.Getenv("MANAGED_CLUSTER1_CTX"); mcl1Ctx == "" {
		mcl1Ctx = "kind-" + mcl1Name
	}
	if kubeConfigPath = os.Getenv("KUBECONFIG"); kubeConfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeConfigPath = filepath.Join(home, ".kube", "config")
	}

	hubConfig, err := buildConfigFromFlags(hubCtx, kubeConfigPath)
	if err != nil {
		return nil, err
	}
	mcl1Config, err := buildConfigFromFlags(mcl1Ctx, kubeConfigPath)
	if err != nil {
		return nil, err
	}
	ctx := clusterValues{
		hub: &clusterConfig{
			name:       hubName,
			context:    hubCtx,
			kubeConfig: hubConfig,
		},
		mcl1: &clusterConfig{
			name:       mcl1Name,
			context:    mcl1Ctx,
			kubeConfig: mcl1Config,
		},
	}

	e2eConf := &TestE2eConfig{
		values: &values{
			cv: &ctx,
		},
		version:        version,
		KubeConfigPath: kubeConfigPath,
	}

	// clearenv set the e2e environment from initial state to empty
	clearenv := func() error {
		fmt.Println("cleaning hub...")

		fmt.Println("deleting all clusters on the hub...")
		// delete all clusters
		err = WaitClustersDeleted(e2eConf.Cluster().Hub().KubeConfig())
		if err != nil {
			return err
		}

		fmt.Println("unjoin managedCluster on the spoke cluster...")
		err := e2eConf.Clusteradm().Unjoin(
			"--context", e2eConf.Cluster().ManagedCluster1().Context(),
			"--cluster-name", e2eConf.Cluster().ManagedCluster1().Name(),
			"--purge-operator=false",
		)
		if err != nil {
			return err
		}
		err = WaitNamespaceDeleted(e2eConf.Cluster().ManagedCluster1().KubeConfig(), config.ManagedClusterNamespace)
		if err != nil {
			return err
		}

		fmt.Println("cleaning clusterManager CR...")
		err = e2eConf.Clusteradm().Clean(
			"--context", e2eConf.Cluster().Hub().Context(),
			"--purge-operator=false",
		)
		if err != nil {
			return err
		}

		err = DeleteClusterCSRs(e2eConf.Cluster().Hub().KubeConfig())
		if err != nil {
			return err
		}
		err = WaitNamespaceDeleted(e2eConf.Cluster().Hub().KubeConfig(), config.HubClusterNamespace)
		if err != nil {
			return err
		}

		return nil
	}
	// ClearEnv will unjoin managed cluster1 and clean hub.
	e2eConf.ClearEnv = clearenv

	return e2eConf, nil
}

func (tec *TestE2eConfig) ResetEnv() error {
	// ensure hub is initialized, and token and apiserver is set.
	fmt.Println("ensure hub is initialized...")
	clusterAdm := tec.Clusteradm()
	err := clusterAdm.Init(
		"--context", tec.Cluster().Hub().Context(),
		"--use-bootstrap-token",
		"--wait",
	)
	if err != nil {
		return err
	}

	if clusterAdm.Result() == nil || len(clusterAdm.Result().RawCommand()) == 0 {
		err = tec.Clusteradm().Get("token")
		if err != nil {
			return err
		}
	}

	// ensure managed cluster1 is join-accepted
	fmt.Println("ensure managed cluster1 is join and accepted...")
	err = clusterAdm.Join(
		"--context", tec.Cluster().ManagedCluster1().Context(),
		"--hub-token", clusterAdm.Result().Token(), "--hub-apiserver", clusterAdm.Result().Host(),
		"--cluster-name", tec.Cluster().ManagedCluster1().Name(),
		"--wait",
		"--force-internal-endpoint-lookup",
	)
	if err != nil {
		return err
	}

	err = clusterAdm.Accept(
		"--clusters", tec.Cluster().ManagedCluster1().Name(),
		"--wait",
		"--context", tec.Cluster().Hub().Context(),
	)
	if err != nil {
		return err
	}

	fmt.Println("reset e2e environment finished.")
	return nil
}
