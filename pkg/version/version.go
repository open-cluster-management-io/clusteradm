// Copyright Contributors to the Open Cluster Management project
package version

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/version"
)

var (
	// commitFromGit is a constant representing the source version that
	// generated this build. It should be set during build via -ldflags.
	commitFromGit string
	// versionFromGit is a constant representing the version tag that
	// generated this build. It should be set during build via -ldflags.
	versionFromGit string
	// major version
	majorFromGit string
	// minor version
	minorFromGit string
	// build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
	buildDate string
)

// Get returns the overall codebase version. It's for detecting
// what code a binary was built from.
func Get() version.Info {
	return version.Info{
		Major:      majorFromGit,
		Minor:      minorFromGit,
		GitCommit:  commitFromGit,
		GitVersion: versionFromGit,
		BuildDate:  buildDate,
	}
}

type VersionBundle struct {
	OCM                      string
	AppAddon                 string
	PolicyAddon              string
	MulticlusterControlplane string
}

var defaultBundleVersion = "0.16.2"

func GetDefaultBundleVersion() string {
	return defaultBundleVersion
}

func GetVersionBundle(version string) (VersionBundle, error) {

	// supporting either "x.y.z" or "vx.y.z" format version
	version = strings.TrimPrefix(version, "v")

	versionBundleList := map[string]VersionBundle{}

	// latest
	versionBundleList["latest"] = VersionBundle{
		OCM:                      "latest",
		AppAddon:                 "latest",
		PolicyAddon:              "latest",
		MulticlusterControlplane: "latest",
	}

	// predefined bundle version
	// TODO: automated version tracking
	versionBundleList["0.14.0"] = VersionBundle{
		OCM:                      "v0.14.0",
		AppAddon:                 "v0.14.0",
		PolicyAddon:              "v0.14.0",
		MulticlusterControlplane: "v0.5.0",
	}

	versionBundleList["0.15.0"] = VersionBundle{
		OCM:                      "v0.15.0",
		AppAddon:                 "v0.15.0",
		PolicyAddon:              "v0.15.0",
		MulticlusterControlplane: "v0.6.0",
	}

	versionBundleList["0.15.2"] = VersionBundle{
		OCM:                      "v0.15.2",
		AppAddon:                 "v0.15.0",
		PolicyAddon:              "v0.15.0",
		MulticlusterControlplane: "v0.6.0",
	}

	versionBundleList["0.16.0"] = VersionBundle{
		OCM:                      "v0.16.0",
		AppAddon:                 "v0.16.0",
		PolicyAddon:              "v0.16.0",
		MulticlusterControlplane: "v0.7.0",
	}

	versionBundleList["0.16.1"] = VersionBundle{
		OCM:                      "v0.16.1",
		AppAddon:                 "v0.16.0",
		PolicyAddon:              "v0.16.0",
		MulticlusterControlplane: "v0.7.0",
	}

	versionBundleList["0.16.2"] = VersionBundle{
		OCM:                      "v0.16.2",
		AppAddon:                 "v0.16.0",
		PolicyAddon:              "v0.16.0",
		MulticlusterControlplane: "v0.7.0",
	}

	// default
	versionBundleList["default"] = versionBundleList[defaultBundleVersion]

	if val, ok := versionBundleList[version]; ok {
		return val, nil
	}
	return VersionBundle{}, fmt.Errorf("couldn't find the requested version bundle: %v", version)
}
