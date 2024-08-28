// Copyright Contributors to the Open Cluster Management project
package scenario

import (
	"embed"
)

//go:embed init
var Files embed.FS

type BundleVersion struct {
	// registration image version
	RegistrationImageVersion string
	// placement image version
	PlacementImageVersion string
	// work image version
	WorkImageVersion string
	// operator image version
	OperatorImageVersion string
	// addon manager image version
	AddonManagerImageVersion string
}
