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
	Registration             string
	Placement                string
	Work                     string
	Operator                 string
	AppAddon                 string
	PolicyAddon              string
	AddonManager             string
	MulticlusterControlplane string
}

var defaultBundleVersion = "0.14.0"

func GetDefaultBundleVersion() string {
	return defaultBundleVersion
}

func GetVersionBundle(version string) (VersionBundle, error) {

	// supporting either "x.y.z" or "vx.y.z" format version
	version = strings.TrimPrefix(version, "v")

	versionBundleList := map[string]VersionBundle{}

	// latest
	versionBundleList["latest"] = VersionBundle{
		Registration:             "latest",
		Placement:                "latest",
		Work:                     "latest",
		Operator:                 "latest",
		AddonManager:             "latest",
		AppAddon:                 "latest",
		PolicyAddon:              "latest",
		MulticlusterControlplane: "latest",
	}

	// predefined bundle version
	// TODO: automated version tracking
	versionBundleList["0.12.0"] = VersionBundle{
		Registration:             "v0.12.0",
		Placement:                "v0.12.0",
		Work:                     "v0.12.0",
		Operator:                 "v0.12.0",
		AddonManager:             "v0.12.0",
		AppAddon:                 "v0.12.0",
		PolicyAddon:              "v0.12.0",
		MulticlusterControlplane: "v0.3.0",
	}

	versionBundleList["0.13.0"] = VersionBundle{
		Registration:             "v0.13.0",
		Placement:                "v0.13.0",
		Work:                     "v0.13.0",
		Operator:                 "v0.13.0",
		AddonManager:             "v0.13.0",
		AppAddon:                 "v0.13.0",
		PolicyAddon:              "v0.13.0",
		MulticlusterControlplane: "v0.4.0",
	}

	versionBundleList["0.13.1"] = VersionBundle{
		Registration:             "v0.13.1",
		Placement:                "v0.13.1",
		Work:                     "v0.13.1",
		Operator:                 "v0.13.1",
		AddonManager:             "v0.13.1",
		AppAddon:                 "v0.13.0",
		PolicyAddon:              "v0.13.0",
		MulticlusterControlplane: "v0.4.0",
	}

	versionBundleList["0.13.2"] = VersionBundle{
		Registration:             "v0.13.2",
		Placement:                "v0.13.2",
		Work:                     "v0.13.2",
		Operator:                 "v0.13.2",
		AddonManager:             "v0.13.2",
		AppAddon:                 "v0.13.0",
		PolicyAddon:              "v0.13.0",
		MulticlusterControlplane: "v0.4.0",
	}

	versionBundleList["0.13.3"] = VersionBundle{
		Registration:             "v0.13.3",
		Placement:                "v0.13.3",
		Work:                     "v0.13.3",
		Operator:                 "v0.13.3",
		AddonManager:             "v0.13.3",
		AppAddon:                 "v0.13.0",
		PolicyAddon:              "v0.13.0",
		MulticlusterControlplane: "v0.4.0",
	}

	versionBundleList["0.14.0"] = VersionBundle{
		Registration:             "v0.14.0",
		Placement:                "v0.14.0",
		Work:                     "v0.14.0",
		Operator:                 "v0.14.0",
		AddonManager:             "v0.14.0",
		AppAddon:                 "v0.14.0",
		PolicyAddon:              "v0.14.0",
		MulticlusterControlplane: "v0.5.0",
	}

	// default
	versionBundleList["default"] = versionBundleList[defaultBundleVersion]

	if val, ok := versionBundleList[version]; ok {
		return val, nil
	}
	return VersionBundle{}, fmt.Errorf("couldn't find the requested version bundle: %v", version)
}
