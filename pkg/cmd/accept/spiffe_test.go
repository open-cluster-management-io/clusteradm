// Copyright Contributors to the Open Cluster Management project
package accept

import (
	"io"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	clusterv1 "open-cluster-management.io/api/cluster/v1"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

func TestSPIFFEAcceptance(t *testing.T) {
	pinned := &clusterv1.ManagedCluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster1",
		Annotations: map[string]string{spiffeTrustDomainAnnotation: "east.example.org"}}}
	plain := &clusterv1.ManagedCluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster2"}}

	plain.ResourceVersion = "42"
	cases := []struct {
		name       string
		mc         *clusterv1.ManagedCluster
		td         string
		pendingCSR bool
		spiffe     bool
		wantPatch  bool
		wantErr    string
	}{
		{"pinned cluster: no CSR", pinned, "", false, true, false, ""},
		{"csr cluster", plain, "", true, false, false, ""},
		{"same trust domain", pinned, "east.example.org", false, true, false, ""},
		{"other trust domain refused", pinned, "west.example.org", false, true, false, "registered from trust domain east.example.org"},
		{"unpinned cluster gets the pin", plain, "east.example.org", false, true, true, ""},
		{"an unpinned cluster with a pending CSR is a csr cluster: no pin", plain, "east.example.org", true, false, false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			isSPIFFE, patch, err := spiffeAcceptance(c.mc, c.td, c.pendingCSR)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Errorf("error %v, want %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if isSPIFFE != c.spiffe || (patch != nil) != c.wantPatch {
				t.Errorf("spiffe %v patch %s", isSPIFFE, patch)
			}
			if c.wantPatch && (!strings.Contains(string(patch), `"open-cluster-management.io/spiffe-trust-domain":"east.example.org"`) ||
				!strings.Contains(string(patch), `"resourceVersion":"42"`)) {
				t.Errorf("patch %s, want the pin guarded by the resourceVersion read", patch)
			}
		})
	}
}

func TestAcceptSPIFFEFlag(t *testing.T) {
	// Only the flags are read; the kubeconfig is never loaded.
	f := cmdutil.NewFactory(genericclioptions.NewConfigFlags(false))
	cmd := NewCmd(genericclioptionsclusteradm.NewClusteradmFlags(f), genericiooptions.IOStreams{Out: io.Discard, ErrOut: io.Discard})
	if cmd.Flags().Lookup("spiffe-trust-domain") == nil {
		t.Error("no --spiffe-trust-domain flag")
	}
}
