// Copyright Contributors to the Open Cluster Management project
package join

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

const testHubID = "spiffe://ocm-hub.example.org/ocm/hub/grpc-server"

// offlineKubeconfig points at a server nothing listens on: join only warns that it can't read the
// cluster's API server URL.
func offlineKubeconfig(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "kubeconfig")
	content := `apiVersion: v1
kind: Config
clusters:
- name: c
  cluster:
    server: https://127.0.0.1:1
contexts:
- name: c
  context:
    cluster: c
    user: u
current-context: c
users:
- name: u
  user:
    token: not-a-hub-token
`
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func spiffeJoinOptions(t *testing.T) (*Options, *cobra.Command, string) {
	t.Helper()
	kubeconfig := offlineKubeconfig(t)
	cf := genericclioptions.NewConfigFlags(false)
	cf.KubeConfig = &kubeconfig
	flags := genericclioptionsclusteradm.NewClusteradmFlags(cmdutil.NewFactory(cf))
	flags.DryRun = true
	cmd := NewCmd(flags, genericiooptions.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}})
	o := newOptions(flags, genericiooptions.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}})
	o.registrationAuth = operatorv1.SPIFFEAuthType
	o.clusterName = "cluster1"
	o.registry = "quay.io/open-cluster-management"
	o.bundleVersion = "default"
	o.mode = "default"
	o.addonKubeClientRegistrationAuth = "csr"
	o.spiffe = spiffeOptions{hubGRPCEndpoint: "ocm-grpc.hub.internal:443", hubSPIFFEID: testHubID}
	// Only hosted mode reads it; the hub in-cluster lookup flag skips dialing it.
	o.managedKubeconfigFile = ""
	o.forceHubInClusterEndpointLookup = true
	return o, cmd, kubeconfig
}

func TestSPIFFEJoinModes(t *testing.T) {
	cases := []struct {
		name      string
		mode      string
		singleton bool
		want      operatorv1.InstallMode
	}{
		{"default", "default", false, operatorv1.InstallModeDefault},
		{"singleton", "default", true, operatorv1.InstallModeSingleton},
		{"hosted", "hosted", false, operatorv1.InstallModeHosted},
		{"singleton hosted", "hosted", true, operatorv1.InstallModeSingletonHosted},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o, cmd, kubeconfig := spiffeJoinOptions(t)
			o.mode, o.singleton = c.mode, c.singleton
			if c.mode == "hosted" {
				o.managedKubeconfigFile = kubeconfig
			}
			if err := o.complete(cmd, nil); err != nil {
				t.Fatal(err)
			}
			if err := o.validate(context.TODO()); err != nil {
				t.Fatal(err)
			}
			k := o.klusterletChartConfig.Klusterlet
			if k.Mode != c.want {
				t.Errorf("mode %s, want %s", k.Mode, c.want)
			}
			rc := k.RegistrationConfiguration
			if rc.RegistrationDriver.AuthType != "spiffe" || rc.RegistrationDriver.SPIFFE == nil ||
				rc.RegistrationDriver.SPIFFE.HubEndpoint != "ocm-grpc.hub.internal:443" ||
				rc.RegistrationDriver.SPIFFE.HubSPIFFEID != testHubID {
				t.Errorf("registration driver %+v", rc.RegistrationDriver)
			}
			if rc.AddOnKubeClientRegistrationDriver == nil || rc.AddOnKubeClientRegistrationDriver.AuthType != "spiffe" {
				t.Errorf("add-on driver %+v", rc.AddOnKubeClientRegistrationDriver)
			}
			if !hasGate(rc.FeatureGates) || !hasGate(k.WorkConfiguration.FeatureGates) {
				t.Errorf("SPIFFERegistration not enabled: %+v %+v", rc.FeatureGates, k.WorkConfiguration.FeatureGates)
			}
			if o.klusterletChartConfig.BootstrapHubKubeConfig != "" || o.klusterletChartConfig.GRPCConfig != "" {
				t.Error("a hub credential was set")
			}

			// The rendered resources have no bootstrap Secret and nothing with a token.
			_, raw, err := chart.RenderKlusterletChart(context.TODO(), o.klusterletChartConfig, OperatorNamespace)
			if err != nil {
				t.Fatal(err)
			}
			all := string(bytes.Join(raw, []byte("\n---\n")))
			// Hosted mode keeps the admin's kubeconfig of the managed cluster, as for every driver;
			// it isn't a hub credential.
			allowed := []string{"image-pull-credentials"}
			if c.mode == "hosted" {
				allowed = append(allowed, "external-managed-kubeconfig")
			}
			for _, obj := range raw {
				if strings.Contains(string(obj), "kind: Secret") && !containsAny(string(obj), allowed) {
					t.Errorf("rendered a Secret other than the image pull credentials:\n%s", obj)
				}
			}
			if strings.Contains(all, "not-a-hub-token") {
				t.Error("the rendered klusterlet has a token")
			}
			if !strings.Contains(all, "authType: spiffe") || !strings.Contains(all, "clusterName: \"cluster1\"") &&
				!strings.Contains(all, "clusterName: cluster1") {
				t.Errorf("rendered Klusterlet misses the spiffe settings:\n%s", all)
			}
		})
	}
}

func hasGate(gates []operatorv1.FeatureGate) bool {
	for _, g := range gates {
		if g.Feature == "SPIFFERegistration" && g.Mode == operatorv1.FeatureGateModeTypeEnable {
			return true
		}
	}
	return false
}

func TestSPIFFEJoinErrors(t *testing.T) {
	cases := []struct {
		name   string
		modify func(o *Options, cmd *cobra.Command)
		want   string
	}{
		{"no hub SPIFFE ID", func(o *Options, _ *cobra.Command) { o.spiffe.hubSPIFFEID = "" }, "--hub-spiffe-id"},
		{"no hub endpoint", func(o *Options, _ *cobra.Command) { o.spiffe.hubGRPCEndpoint = "" }, "--hub-grpc-endpoint"},
		{"a token", func(o *Options, _ *cobra.Command) { o.token = "x" }, "--hub-token is not used with spiffe"},
		{"a hub API server", func(o *Options, _ *cobra.Command) { o.hubAPIServer = "https://hub:6443" }, "not used with spiffe"},
		{"no cluster name", func(o *Options, _ *cobra.Command) { o.clusterName = "" }, "--cluster-name is required with spiffe"},
		{"bad hub ID", func(o *Options, _ *cobra.Command) { o.spiffe.hubSPIFFEID = "https://hub" }, "--hub-spiffe-id"},
		{"template typo", func(o *Options, _ *cobra.Command) {
			o.spiffe.agentIDTemplate = "spiffe://{trustDomain}/ocm/{clutser}/{role}"
		}, "clutser"},
		{"token add-on driver", func(o *Options, cmd *cobra.Command) {
			_ = cmd.Flags().Set("addon-kubeclient-registration-auth", "token")
			o.addonKubeClientRegistrationAuth = "token"
		}, "not supported"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o, cmd, _ := spiffeJoinOptions(t)
			c.modify(o, cmd)
			err := o.complete(cmd, nil)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %v, want %q", err, c.want)
			}
		})
	}
}

func TestSPIFFEJoinTemplates(t *testing.T) {
	tpl := "spiffe://{trustDomain}/k8s/{cluster}/ns/{namespace}/ocm/{role}"
	o, cmd, _ := spiffeJoinOptions(t)
	o.spiffe.agentIDTemplate = tpl
	o.spiffe.trustDomain = "east.example.org"
	if err := o.complete(cmd, nil); err != nil {
		t.Fatal(err)
	}
	s := o.klusterletChartConfig.Klusterlet.RegistrationConfiguration.RegistrationDriver.SPIFFE
	if s.AgentIDTemplate != tpl || s.TrustDomain != "east.example.org" {
		t.Errorf("spiffe config %+v", s)
	}
}

func TestSPIFFENextSteps(t *testing.T) {
	o, _, _ := spiffeJoinOptions(t)
	o.klusterletChartConfig.Klusterlet.ClusterName = "cluster1"
	o.klusterletChartConfig.Klusterlet.Name = "klusterlet"
	out := &bytes.Buffer{}
	o.printSPIFFENextSteps(out, &operatorv1.Klusterlet{Status: operatorv1.KlusterletStatus{SPIFFE: &operatorv1.KlusterletSPIFFEStatus{
		AgentIDs: []operatorv1.SPIFFEAgentID{
			{Role: "registration", SPIFFEID: "spiffe://ocm-hub.example.org/ocm/cluster/cluster1/agent/registration"},
			{Role: "work", SPIFFEID: "spiffe://ocm-hub.example.org/ocm/cluster/cluster1/agent/work"},
		}}}})
	for _, want := range []string{"registration  spiffe://ocm-hub.example.org/ocm/cluster/cluster1/agent/registration",
		"work          spiffe://ocm-hub.example.org/ocm/cluster/cluster1/agent/work", "accept --clusters cluster1"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output misses %q:\n%s", want, out)
		}
	}
	out.Reset()
	o.printSPIFFENextSteps(out, nil)
	if !strings.Contains(out.String(), "SPIFFEAdmissionReady") || !strings.Contains(out.String(), "status.spiffe.agentIDs") {
		t.Errorf("--wait=false output:\n%s", out)
	}
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
