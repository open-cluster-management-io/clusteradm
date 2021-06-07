// Copyright Contributors to the Open Cluster Management project
package scenario

import (
	"embed"

	"open-cluster-management.io/clusteradm/pkg/helpers/asset"
)

//go:embed musttemplateasset
var files embed.FS

func GetScenarioResourcesReader() *asset.ScenarioResourcesReader {
	return asset.NewScenarioResourcesReader(&files)
}
