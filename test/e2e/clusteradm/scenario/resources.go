// Copyright Contributors to the Open Cluster Management project
package init

import (
	"embed"

	"github.com/stolostron/applier/pkg/asset"
)

//go:embed init
var files embed.FS

func GetScenarioResourcesReader() *asset.ScenarioResourcesReader {
	return asset.NewScenarioResourcesReader(&files)
}
