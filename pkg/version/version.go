// Copyright Contributors to the Open Cluster Management project
package version

import (
	"encoding/json"
	"fmt"
	"os"
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
	OCM                      string `json:"ocm"`
	AppAddon                 string `json:"app_addon"`
	PolicyAddon              string `json:"policy_addon"`
	MulticlusterControlplane string `json:"multicluster_controlplane"`
}

var defaultBundleVersion = "0.16.1"

func GetDefaultBundleVersion() string {
	return defaultBundleVersion
}

// GetVersionBundleFromFile reads a version bundle from a file
func GetVersionBundleFromFile(filePath string) (VersionBundle, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return VersionBundle{}, fmt.Errorf("failed to read version bundle file: %w", err)
	}

	var bundle VersionBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return VersionBundle{}, fmt.Errorf("failed to unmarshal version bundle: %w", err)
	}

	var missingKeys []string
	if bundle.OCM == "" {
		missingKeys = append(missingKeys, "ocm")
	}
	if bundle.AppAddon == "" {
		missingKeys = append(missingKeys, "app_addon")
	}
	if bundle.PolicyAddon == "" {
		missingKeys = append(missingKeys, "policy_addon")
	}
	if bundle.MulticlusterControlplane == "" {
		missingKeys = append(missingKeys, "multicluster_controlplane")
	}
	if len(missingKeys) > 0 {
		return VersionBundle{}, fmt.Errorf(
			"invalid version bundle file: missing required keys: [%s]",
			strings.Join(missingKeys, ", "),
		)
	}

	return bundle, nil
}

func GetVersionBundle(version string, versionBundleFile string) (VersionBundle, error) {
	// If version bundle file is provided, read from it
	if versionBundleFile != "" {
		return GetVersionBundleFromFile(versionBundleFile)
	}

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

	// default
	versionBundleList["default"] = versionBundleList[defaultBundleVersion]

	if val, ok := versionBundleList[version]; ok {
		return val, nil
	}
	return VersionBundle{}, fmt.Errorf("couldn't find the requested version bundle: %v", version)
}
