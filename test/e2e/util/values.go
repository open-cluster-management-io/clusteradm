// Copyright Contributors to the Open Cluster Management project
package util

type clusterConfig struct {
	name    string
	context string
}

func (cc *clusterConfig) Name() string {
	return cc.name
}

func (cc *clusterConfig) Context() string {
	return cc.context
}

type clusterValues struct {
	hub  *clusterConfig
	mcl1 *clusterConfig
	mcl2 *clusterConfig
}

func (cv *clusterValues) Hub() *clusterConfig {
	return cv.hub
}

func (cv *clusterValues) ManagedCluster1() *clusterConfig {
	return cv.mcl1
}

func (cv *clusterValues) ManagedCluster2() *clusterConfig {
	return cv.mcl2
}

type values struct {
	// cv stores the clusters infomation
	cv *clusterValues
}
