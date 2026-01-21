// Copyright Contributors to the Open Cluster Management project
package util

import "k8s.io/client-go/rest"

type TestE2eConfig struct {
	values  *values
	version string

	KubeConfigPath string

	ClearEnv func() error
}

func (tec *TestE2eConfig) Cluster() *clusterValues {
	return tec.values.cv
}

func (tec *TestE2eConfig) Clusteradm() clusteradmInterface {
	return &clusteradm{version: tec.version}
}

func (tec *TestE2eConfig) HubKubeConfig() *rest.Config { return tec.values.cv.Hub().kubeConfig }

func (tec *TestE2eConfig) ManagedClusterKubeConfig() *rest.Config {
	return tec.values.cv.ManagedCluster1().kubeConfig
}
