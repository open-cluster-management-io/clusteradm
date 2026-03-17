// Copyright Contributors to the Open Cluster Management project
package version

import (
	"reflect"
	"runtime/debug"
	"testing"
)

func TestVersionFromBuildInfo(t *testing.T) {
	tests := []struct {
		name           string
		info           *debug.BuildInfo
		fallbackCommit string
		wantVersion    string
		wantCommit     string
	}{
		{
			name: "tagged release via go install",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.2.0"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc1234def5678"},
					{Key: "vcs.modified", Value: "false"},
				},
			},
			wantVersion: "v1.2.0",
			wantCommit:  "abc1234",
		},
		{
			name: "tagged release with dirty tree",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.2.0"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc1234def5678"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			wantVersion: "v1.2.0-dirty",
			wantCommit:  "abc1234",
		},
		{
			name: "(devel) version is ignored",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc1234def5678"},
					{Key: "vcs.modified", Value: "false"},
				},
			},
			wantVersion: "",
			wantCommit:  "abc1234",
		},
		{
			name: "(devel) with dirty tree does not append -dirty",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc1234def5678"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			wantVersion: "",
			wantCommit:  "abc1234",
		},
		{
			name: "empty version is ignored",
			info: &debug.BuildInfo{
				Main:     debug.Module{Version: ""},
				Settings: []debug.BuildSetting{},
			},
			wantVersion: "",
			wantCommit:  "",
		},
		{
			name: "short vcs.revision is ignored",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.0.0"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc"},
				},
			},
			wantVersion: "v1.0.0",
			wantCommit:  "",
		},
		{
			name:           "fallback commit is preserved when vcs.revision present",
			fallbackCommit: "existing",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.0.0"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc1234def5678"},
				},
			},
			wantVersion: "v1.0.0",
			wantCommit:  "existing",
		},
		{
			name: "no vcs settings",
			info: &debug.BuildInfo{
				Main:     debug.Module{Version: "v0.10.1"},
				Settings: []debug.BuildSetting{},
			},
			wantVersion: "v0.10.1",
			wantCommit:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVersion, gotCommit := versionFromBuildInfo(tt.info, tt.fallbackCommit)
			if gotVersion != tt.wantVersion {
				t.Errorf("version = %q, want %q", gotVersion, tt.wantVersion)
			}
			if gotCommit != tt.wantCommit {
				t.Errorf("commit = %q, want %q", gotCommit, tt.wantCommit)
			}
		})
	}
}

func TestGetVersionBundle(t *testing.T) {
	expectedVersionBundle := VersionBundle{
		OCM:         "v1.1.1",
		PolicyAddon: "v0.16.0",
	}

	tests := []struct {
		name                  string
		version               string
		versionBundleFile     string
		expectedVersionBundle func() VersionBundle
		wantErr               bool
	}{
		{
			name:              "default",
			version:           "default",
			versionBundleFile: "",
			expectedVersionBundle: func() VersionBundle {
				versionBundle, _ := getVersionBundle("default")
				return versionBundle
			},
		},
		{
			name:              "specific version",
			version:           "v1.1.0",
			versionBundleFile: "",
			expectedVersionBundle: func() VersionBundle {
				b := expectedVersionBundle
				b.OCM = "v1.1.0"
				return b
			},
		},
		{
			name:                  "override",
			version:               "v1.1.0",
			versionBundleFile:     "testdata/bundle-overrides.json",
			expectedVersionBundle: func() VersionBundle { return expectedVersionBundle },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bundle, err := GetVersionBundle(test.version, test.versionBundleFile)
			if (err != nil) != test.wantErr {
				t.Errorf("GetVersionBundle() error = %v, wantErr %v", err, test.wantErr)
			}
			expected := test.expectedVersionBundle()
			if !reflect.DeepEqual(bundle, expected) {
				t.Errorf("GetVersionBundle() = %v, want %v", bundle, expected)
			}
		})
	}
}
