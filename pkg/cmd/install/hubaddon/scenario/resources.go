// Copyright Contributors to the Open Cluster Management project
package scenario

import (
	"embed"

	"github.com/stolostron/applier/pkg/asset"
)

//go:embed addon
var files embed.FS

func GetScenarioResourcesReader() *asset.ScenarioResourcesReader {
	return asset.NewScenarioResourcesReader(&files)
}
