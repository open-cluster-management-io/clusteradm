// Copyright Contributors to the Open Cluster Management project
package scenario

import (
	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers/asset"
)

func GetScenarioResourcesReader() *asset.ScenarioResourcesReader {
	return scenario.GetScenarioResourcesReader()
}
