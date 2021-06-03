// Copyright Contributors to the Open Cluster Management project
package scenario

import (
	"embed"

	"open-cluster-management.io/clusteradm/pkg/helpers"
)

//go:embed init
var files embed.FS

func GetScenarioResourcesReader() *helpers.ScenarioResourcesReader {
	return helpers.NewScenarioResourcesReader(&files)
}
