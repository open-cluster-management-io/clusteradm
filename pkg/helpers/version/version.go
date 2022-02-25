// Copyright Contributors to the Open Cluster Management project
package version

import (
	"fmt"
	"strings"
)

type VersionBundle struct {
	Registration string
	Placement    string
	Work         string
	Operator     string
}

func GetVersionBundle(version string) (VersionBundle, error) {

	// supporting either "x.y.z" or "vx.y.z" format version
	if strings.HasPrefix(version, "v") {
		version = strings.TrimPrefix(version, "v")
	}

	versionBundleList := map[string]VersionBundle{}

	// latest
	versionBundleList["latest"] = VersionBundle{
		Registration: "latest",
		Placement:    "latest",
		Work:         "latest",
		Operator:     "latest",
	}

	// predifined bundle version
	// TODO: automated version tracking
	versionBundleList["0.5.0"] = VersionBundle{
		Registration: "v0.5.0",
		Placement:    "v0.2.0",
		Work:         "v0.5.0",
		Operator:     "v0.5.0",
	}

	versionBundleList["0.6.0"] = VersionBundle{
		Registration: "v0.6.0",
		Placement:    "v0.3.0",
		Work:         "v0.6.0",
		Operator:     "v0.6.0",
	}

	// default
	versionBundleList["default"] = versionBundleList["0.6.0"]

	if val, ok := versionBundleList[version]; ok {
		return val, nil
	}
	return VersionBundle{}, fmt.Errorf("couldn't find the requested version bundle: %v", version)
}
