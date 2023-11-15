// Copyright Contributors to the Open Cluster Management project
package util

import "k8s.io/client-go/rest"

type clusterConfig struct {
	name       string
	context    string
	kubeConfig *rest.Config
}

func (cc *clusterConfig) Name() string {
	return cc.name
}

func (cc *clusterConfig) Context() string {
	return cc.context
}

func (cc *clusterConfig) KubeConfig() *rest.Config {
	return cc.kubeConfig
}

type clusterValues struct {
	hub  *clusterConfig
	mcl1 *clusterConfig
}

func (cv *clusterValues) Hub() *clusterConfig {
	return cv.hub
}

func (cv *clusterValues) ManagedCluster1() *clusterConfig {
	return cv.mcl1
}

type values struct {
	// cv stores the clusters infomation
	cv *clusterValues
}
