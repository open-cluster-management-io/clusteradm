// Copyright Contributors to the Open Cluster Management project
package helm

import (
	"encoding/base64"
	"io"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

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

// The release name is taken, but Helm cannot give a status for it.
func TestReleaseStatusUnreadableHistory(t *testing.T) {
	const name = "test-release"
	tests := []struct {
		name string
		data string
	}{
		// Helm skips Secrets it cannot decode and returns an empty history.
		{name: "undecodable", data: "not a release"},
		{name: "no info", data: base64.StdEncoding.EncodeToString([]byte(`{"name":"test-release","version":1}`))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sh.helm.release.v1." + name + ".v1",
					Namespace: "default",
					Labels:    map[string]string{"owner": "helm", "name": name},
				},
				Data: map[string][]byte{"release": []byte(tt.data)},
			}
			cfg := &action.Configuration{
				Releases:   storage.Init(driver.NewSecrets(fake.NewClientset(secret).CoreV1().Secrets("default"))),
				KubeClient: &kubefake.PrintingKubeClient{Out: io.Discard},
				Log:        func(string, ...interface{}) {},
			}

			if _, err := releaseStatus(cfg, name); err == nil {
				t.Fatal("expected an error for a release without a readable revision")
			}
		})
	}
}
