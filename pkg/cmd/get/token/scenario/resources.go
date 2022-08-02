// Copyright Contributors to the Open Cluster Management project
package scenario

import (
	"github.com/stolostron/applier/pkg/asset"
	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
)

func GetScenarioResourcesReader() *asset.ScenarioResourcesReader {
	return scenario.GetScenarioResourcesReader()
}
