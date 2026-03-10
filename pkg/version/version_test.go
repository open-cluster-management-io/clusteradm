// Copyright Contributors to the Open Cluster Management project
package version

import (
	"reflect"
	"testing"
)

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
