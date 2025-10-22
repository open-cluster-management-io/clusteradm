// Copyright Contributors to the Open Cluster Management project
package version

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/version"
	"k8s.io/klog/v2"
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
	OCM                      string `json:"ocm"`
	AppAddon                 string `json:"app_addon"`
	PolicyAddon              string `json:"policy_addon"`
	MulticlusterControlplane string `json:"multicluster_controlplane"`
}

var defaultBundleVersion = "1.1.0"

func GetDefaultBundleVersion() string {
	return defaultBundleVersion
}

// GetVersionBundle returns a version bundle for the requested version and optional overrides.
func GetVersionBundle(version string, versionBundleFile string) (VersionBundle, error) {
	bundle, err := getVersionBundle(version)
	if err != nil {
		return VersionBundle{}, err
	}

	if versionBundleFile != "" {
		bundle, err = overrideVersionBundle(bundle, versionBundleFile)
		if err != nil {
			return VersionBundle{}, err
		}
	}

	return bundle, nil
}

func getVersionBundle(version string) (VersionBundle, error) {

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
	versionBundleList["0.16.0"] = VersionBundle{
		OCM:                      "v0.16.0",
		AppAddon:                 "v0.16.0",
		PolicyAddon:              "v0.16.0",
		MulticlusterControlplane: "v0.7.0",
	}

	versionBundleList["1.0.0"] = VersionBundle{
		OCM:                      "v1.0.0",
		AppAddon:                 "v0.16.0",
		PolicyAddon:              "v0.16.0",
		MulticlusterControlplane: "v0.7.0",
	}

	versionBundleList["1.1.0"] = VersionBundle{
		OCM:                      "v1.1.0",
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

func overrideVersionBundle(bundle VersionBundle, filePath string) (VersionBundle, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return VersionBundle{}, fmt.Errorf("failed to read version bundle file: %w", err)
	}
	if err := json.Unmarshal(data, &bundle); err != nil {
		return VersionBundle{}, fmt.Errorf("failed to unmarshal version bundle: %w", err)
	}

	klog.V(3).InfoS("applied overrides to version bundle", "finalBundle", bundle)
	return bundle, nil
}
