// Copyright Contributors to the Open Cluster Management project
package helm

import (
	"io"
	"testing"

	"helm.sh/helm/v3/pkg/action"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
)

func TestReleaseStatus(t *testing.T) {
	const name = "test-release"
	tests := []struct {
		name      string
		revisions []release.Status
		want      release.Status
	}{
		{name: "no release", want: ""},
		{name: "deployed", revisions: []release.Status{release.StatusDeployed}, want: release.StatusDeployed},
		{name: "failed install", revisions: []release.Status{release.StatusFailed}, want: release.StatusFailed},
		{
			name:      "latest revision wins",
			revisions: []release.Status{release.StatusSuperseded, release.StatusFailed},
			want:      release.StatusFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &action.Configuration{
				Releases:   storage.Init(driver.NewMemory()),
				KubeClient: &kubefake.PrintingKubeClient{Out: io.Discard},
				Log:        func(string, ...interface{}) {},
			}
			for i, status := range tt.revisions {
				rel := &release.Release{Name: name, Version: i + 1, Info: &release.Info{Status: status}}
				if err := cfg.Releases.Create(rel); err != nil {
					t.Fatal(err)
				}
			}

			got, err := releaseStatus(cfg, name)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("releaseStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}
