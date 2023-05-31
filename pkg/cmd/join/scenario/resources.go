// Copyright Contributors to the Open Cluster Management project
package scenario

import (
	"embed"
)

//go:embed join singleton bootstrap_hub_kubeconfig.yaml
var Files embed.FS
