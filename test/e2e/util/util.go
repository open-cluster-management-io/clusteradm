// Copyright Contributors to the Open Cluster Management project
package util

import (
	"fmt"
	"os"
	"time"
)

// PrepareE2eEnvironment will init the e2e environment and join managedcluster1 to hub.
func PrepareE2eEnvironment() *TestE2eConfig {
	conf := initE2E()

	conf.ResetEnv()

	return conf
}

// initE2E get environment variables and init e2e environment.
func initE2E() *TestE2eConfig {

	e2eConf := NewTestE2eConfig(
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
			panic(fmt.Sprintf("error occurs while unjoing managedcluster1: %s", err.Error()))
		}

		fmt.Println("cleaning hub...")
		err = e2eConf.Clusteradm().Clean("--context", e2eConf.Cluster().Hub().Context())
		if err != nil {
			panic(fmt.Sprintf("error occurs while cleaning hub: %s", err.Error()))
		}
		// clear token and apiserver value
		e2eConf.clusteradm.h = HandledOutput{}

		// wait for resources to terminate
		t := 30
		fmt.Fprintf(os.Stdout, "wait resources to be terminated for %d seconds", t)
		time.Sleep(time.Duration(t) * time.Second)
	}
	// ClearEnv will unjoin managed cluster1 and clean hub.
	e2eConf.ClearEnv = clearenv

	return e2eConf
}

func (tec *TestE2eConfig) ResetEnv() {
	// ensure hub is initialized, and token and apiserver is set.
	fmt.Println("ensure hub is initilized...")
	err := tec.Clusteradm().Init(
		"--context", tec.Cluster().Hub().Context(),
		"--use-bootstrap-token",
		"--wait",
	)
	if err != nil {
		panic(fmt.Sprintf("error occurs while initializing hub: %s", err))
	}

	if tec.CommandResult() == nil || len(tec.CommandResult().RawCommand()) == 0 {
		err = tec.Clusteradm().Get("token")
		if err != nil {
			panic(fmt.Sprintf("error occurs while get token from hub: %s", err))
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
		panic(fmt.Sprintf("error occurs while managedcluster1 joining hub: %s", err))
	}

	err = tec.Clusteradm().Accept(
		"--clusters", tec.Cluster().ManagedCluster1().Name(),
		"--wait", "30",
		"--context", tec.Cluster().Hub().Context(),
	)
	if err != nil {
		panic(fmt.Sprintf("error occurs while hub accepting managedcluster1: %s", err))
	}

	// ensure managed clsuter2 is not join-accepted
	fmt.Println("ensure managed cluster2 is unjoined...")
	err = tec.Clusteradm().Unjoin(
		"--context", tec.Cluster().ManagedCluster2().Context(),
		"--cluster-name", tec.Cluster().ManagedCluster2().Name(),
	)
	if err != nil {
		// TODO: figure out how to catch this error and then use panic here(when unjoin a unjoined managedcluster, this error occurs)
		// 2022/03/04 06:43:00 the server could not find the requested resource (get appliedmanifestworks.work.open-cluster-management.io)
		fmt.Printf("error occurs while unjoing managedcluster2: %s\n", err.Error())
	}

	fmt.Println("reset e2e environment finished.")
}
