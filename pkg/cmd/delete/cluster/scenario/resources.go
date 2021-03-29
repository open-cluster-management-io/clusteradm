// Copyright Contributors to the Open Cluster Management project
package scenario

import (
	"embed"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"
)

//go:embed delete delete/*/*/_helpers.tpl
var files embed.FS

func GetApplierScenarioResourcesReader() *applierscenarios.ApplierScenarioResourcesReader {
	return applierscenarios.NewApplierScenarioResourcesReader(&files)
}
