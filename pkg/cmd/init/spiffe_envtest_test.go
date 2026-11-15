// Copyright Contributors to the Open Cluster Management project
package init

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/rest"
)

func envInitOptions(t *testing.T, cfg *rest.Config, drivers []string, s spiffeOptions) (*Options, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	o := newOptions(envFlags(cfg), genericiooptions.IOStreams{Out: out, ErrOut: out})
	o.registry = "quay.io/open-cluster-management"
	o.bundleVersion = "default"
	o.registrationDrivers = drivers
	o.output = "text"
	o.resourceQosClass = "Default"
	o.createNamespace = true
	if s.endpointType == "" {
		s.endpointType = "loadBalancer"
	}
	o.spiffe = s
	return o, out
}

// validate runs the preflight checks against the cluster, then the flag checks.
func TestInitValidateAgainstCluster(t *testing.T) {
	cfg := testEnvConfig(t)
	o, _ := envInitOptions(t, cfg, []string{"spiffe", "kerberos"}, spiffeOptions{trustDomain: "ocm-hub.example.org"})
	if err := o.validate(context.TODO()); err == nil || !strings.Contains(err.Error(), "valid drivers") {
		t.Errorf("an unknown driver: %v", err)
	}
	o, _ = envInitOptions(t, cfg, []string{"csr"}, spiffeOptions{trustDomain: "ocm-hub.example.org"})
	if err := o.validate(context.TODO()); err == nil || !strings.Contains(err.Error(), "--spiffe-*") {
		t.Errorf("spiffe flags without the driver: %v", err)
	}
}

// A spiffe-only hub is installed without any bootstrap token or ServiceAccount, and prints a join
// command with no token.
func TestInitSPIFFEOnlyAgainstCluster(t *testing.T) {
	cfg := testEnvConfig(t)
	o, out := envInitOptions(t, cfg, []string{"spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org"})
	if err := o.complete(&cobra.Command{}, nil); err != nil {
		t.Fatal(err)
	}
	if err := o.run(context.TODO()); err != nil {
		t.Fatal(err)
	}
	if o.clusterManagerChartConfig.CreateBootstrapSA || o.clusterManagerChartConfig.CreateBootstrapToken {
		t.Error("a bootstrap credential was requested for a spiffe-only hub")
	}
	if !strings.Contains(out.String(), "--registration-auth spiffe") || strings.Contains(out.String(), "--hub-token") {
		t.Errorf("output:\n%s", out)
	}
}

// With spiffe next to csr (dry run), the token join command is followed by the spiffe one.
func TestInitMixedDriversDryRun(t *testing.T) {
	// A dry run still maps resources through the API server: use the test one, never the
	// default kubeconfig.
	cfg := testEnvConfig(t)
	for _, useBootstrapToken := range []bool{false, true} {
		o, out := envInitOptions(t, cfg, []string{"csr", "spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org"})
		o.ClusteradmFlags.DryRun = true
		o.useBootstrapToken = useBootstrapToken
		if err := o.complete(&cobra.Command{}, nil); err != nil {
			t.Fatal(err)
		}
		if err := o.run(context.TODO()); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "--hub-token") || !strings.Contains(out.String(), "--registration-auth spiffe") {
			t.Errorf("output:\n%s", out)
		}
	}
}

// A values file that turns off both bootstrap credentials gets one back from
// --use-bootstrap-token, as the join token is read from it. (The cluster is unreachable here:
// only the decision before applying matters.)
func TestInitValuesFileDisablesBootstrap(t *testing.T) {
	values := filepath.Join(t.TempDir(), "values.yaml")
	if err := os.WriteFile(values, []byte("createBootstrapSA: false\ncreateBootstrapToken: false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, useBootstrapToken := range []bool{false, true} {
		o, _ := envInitOptions(t, &rest.Config{Host: "https://127.0.0.1:1"}, []string{"csr"}, spiffeOptions{})
		o.clusterManagerValuesFile = values
		o.useBootstrapToken = useBootstrapToken
		if err := o.complete(&cobra.Command{}, nil); err != nil {
			t.Fatal(err)
		}
		_ = o.run(context.TODO()) // fails to apply: nothing listens
		if useBootstrapToken != o.clusterManagerChartConfig.CreateBootstrapToken ||
			useBootstrapToken == o.clusterManagerChartConfig.CreateBootstrapSA {
			t.Errorf("useBootstrapToken %v: SA %v, token %v", useBootstrapToken,
				o.clusterManagerChartConfig.CreateBootstrapSA, o.clusterManagerChartConfig.CreateBootstrapToken)
		}
	}
}
