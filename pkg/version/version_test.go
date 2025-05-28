// Copyright Contributors to the Open Cluster Management project
package version

import (
	"reflect"
	"testing"
)

func TestGetVersionBundle(t *testing.T) {
	expectedVersionBundle := VersionBundle{
		OCM:                      "v0.16.1",
		AppAddon:                 "v0.16.0",
		PolicyAddon:              "v0.16.0",
		MulticlusterControlplane: "v0.7.0",
	}

	tests := []struct {
		name                  string
		version               string
		versionBundleFile     string
		expectedVersionBundle func() VersionBundle
		wantErr               bool
	}{
		{
			name:                  "default",
			version:               "default",
			versionBundleFile:     "",
			expectedVersionBundle: func() VersionBundle { return expectedVersionBundle },
		},
		{
			name:              "specific version",
			version:           "v0.16.0",
			versionBundleFile: "",
			expectedVersionBundle: func() VersionBundle {
				b := expectedVersionBundle
				b.OCM = "v0.16.0"
				return b
			},
		},
		{
			name:                  "override",
			version:               "v0.16.0",
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
