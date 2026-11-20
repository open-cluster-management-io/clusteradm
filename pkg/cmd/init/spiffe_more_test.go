// Copyright Contributors to the Open Cluster Management project
package init

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"

	operatorv1 "open-cluster-management.io/api/operator/v1"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

func TestSPIFFEServerIDErrors(t *testing.T) {
	agentTmpl := "spiffe://{trustDomain}/ocm/cluster/{cluster}/agent/{role}"
	cases := []struct {
		c    operatorv1.SPIFFERegistrationConfig
		want string
	}{
		{operatorv1.SPIFFERegistrationConfig{TrustDomain: "Not A Domain"}, "--spiffe-trust-domain"},
		{operatorv1.SPIFFERegistrationConfig{TrustDomain: "ocm-hub.example.org", ServerSPIFFEID: "https://hub"}, "--spiffe-server-id"},
		{operatorv1.SPIFFERegistrationConfig{TrustDomain: "ocm-hub.example.org",
			ServerSPIFFEID: "spiffe://other.example.org/ocm/hub/grpc-server"}, "not in trust domain"},
		{operatorv1.SPIFFERegistrationConfig{TrustDomain: "ocm-hub.example.org",
			AgentIDTemplate: "spiffe://{trustDomain}/ocm/{cluster}/{clutser}"}, "clutser"},
		{operatorv1.SPIFFERegistrationConfig{TrustDomain: "ocm-hub.example.org", AgentIDTemplate: agentTmpl,
			ServerSPIFFEID: "spiffe://ocm-hub.example.org/ocm/cluster/hub/agent/work"}, "server"},
	}
	for _, c := range cases {
		if _, err := spiffeServerID(&c.c); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%+v: error %v, want %q", c.c, err, c.want)
		}
	}
}

func TestSPIFFEEndpointErrors(t *testing.T) {
	o, _ := spiffeInitOptions(t, []string{"spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org", endpointType: "hostname"})
	if _, err := o.spiffeEndpoint(); err == nil || !strings.Contains(err.Error(), "--spiffe-server is required") {
		t.Errorf("hostname without server: %v", err)
	}
	if err := o.applySPIFFE(); err == nil {
		t.Error("applySPIFFE accepted hostname without a server")
	}
	o.spiffe.server = "ocm-grpc.hub.example.com"
	if e, err := o.spiffeEndpoint(); err != nil || e.GRPC.Hostname.Host != "ocm-grpc.hub.example.com" {
		t.Errorf("hostname: %+v, %v", e, err)
	}
	o.spiffe.endpointType = "nodePort"
	if _, err := o.spiffeEndpoint(); err == nil || !strings.Contains(err.Error(), "unknown --spiffe-endpoint-type") {
		t.Errorf("unknown type: %v", err)
	}
}

func TestValidateSPIFFEFlagsWithoutDriver(t *testing.T) {
	o, _ := spiffeInitOptions(t, []string{"csr"}, spiffeOptions{trustDomain: "ocm-hub.example.org"})
	if err := o.validateSPIFFEFlags(); err == nil {
		t.Error("spiffe flags accepted without the spiffe driver")
	}
	o.spiffe = spiffeOptions{endpointType: "loadBalancer"}
	if err := o.validateSPIFFEFlags(); err != nil {
		t.Errorf("no spiffe flags: %v", err)
	}
	if err := o.applySPIFFE(); err != nil {
		t.Errorf("applySPIFFE without the driver: %v", err)
	}
}

func TestEnableFeatureExisting(t *testing.T) {
	gates := enableFeature([]operatorv1.FeatureGate{{Feature: "SPIFFERegistration", Mode: operatorv1.FeatureGateModeTypeDisable}},
		"SPIFFERegistration")
	if len(gates) != 1 || gates[0].Mode != operatorv1.FeatureGateModeTypeEnable {
		t.Errorf("gates %+v", gates)
	}
}

func TestSPIFFEHubEndpoint(t *testing.T) {
	o, _ := spiffeInitOptions(t, []string{"spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org"})
	svc := func(ingress corev1.LoadBalancerIngress, ports ...corev1.ServicePort) *corev1.Service {
		return &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: spiffeServiceName, Namespace: spiffeServiceNamespace},
			Spec:   corev1.ServiceSpec{Ports: ports},
			Status: corev1.ServiceStatus{LoadBalancer: corev1.LoadBalancerStatus{Ingress: []corev1.LoadBalancerIngress{ingress}}}}
	}
	cases := []struct {
		name string
		svc  *corev1.Service
		want string
	}{
		{"no Service yet", nil, ""},
		{"a hostname", svc(corev1.LoadBalancerIngress{Hostname: "lb.example.com"}, corev1.ServicePort{Name: "grpc", Port: 8443}), "lb.example.com:8443"},
		{"an IP", svc(corev1.LoadBalancerIngress{IP: "10.0.0.1"}, corev1.ServicePort{Name: "metrics", Port: 9090}), "10.0.0.1:443"},
		{"no address yet", svc(corev1.LoadBalancerIngress{}), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kube := kubefake.NewSimpleClientset()
			if c.svc != nil {
				kube = kubefake.NewSimpleClientset(c.svc)
			}
			if got := o.spiffeHubEndpoint(context.TODO(), kube); got != c.want {
				t.Errorf("endpoint %q, want %q", got, c.want)
			}
		})
	}
}

func TestPrintSPIFFEJoinOptions(t *testing.T) {
	o, _ := spiffeInitOptions(t, []string{"spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org",
		addOnAgentIDTemplate: "spiffe://{trustDomain}/k8s/{cluster}/{addon}/{agent}"})
	if err := o.complete(&cobra.Command{}, nil); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	if err := o.printSPIFFEJoin(context.TODO(), out, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "--spiffe-addon-agent-id-template") || !strings.Contains(out.String(), "no address yet") {
		t.Errorf("output:\n%s", out)
	}
}

type rawBytes []byte

func (r rawBytes) RawAppliedResources() []byte { return r }

func TestFinishSPIFFEOnly(t *testing.T) {
	o, out := spiffeInitOptions(t, []string{"spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org", server: "hub.example.com"})
	if err := o.complete(&cobra.Command{}, nil); err != nil {
		t.Fatal(err)
	}
	o.outputFile = filepath.Join(t.TempDir(), "out.yaml")
	o.outputJoinCommandFile = filepath.Join(t.TempDir(), "join.sh")
	if err := o.finishSPIFFEOnly(context.TODO(), rawBytes("kind: ClusterManager"), nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "hub.example.com:443") {
		t.Errorf("output:\n%s", out)
	}
	// The join command file, as a script of the cluster name and the admission enforcement.
	script, err := os.ReadFile(o.outputJoinCommandFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--cluster-name $1", "--spiffe-admission-enforcement $2", "--hub-grpc-endpoint hub.example.com:443",
		"--hub-spiffe-id spiffe://ocm-hub.example.org/ocm/hub/grpc-server"} {
		if !strings.Contains(string(script), want) {
			t.Errorf("join command file misses %q:\n%s", want, script)
		}
	}
	o.outputJoinCommandFile = filepath.Join(t.TempDir(), "missing", "join.sh")
	if err := o.finishSPIFFEOnly(context.TODO(), rawBytes("x"), nil); err == nil {
		t.Error("no error writing the join command file to a missing directory")
	}
	o.outputJoinCommandFile = ""
	o.outputFile = filepath.Join(t.TempDir(), "missing", "out.yaml")
	if err := o.finishSPIFFEOnly(context.TODO(), rawBytes("x"), nil); err == nil {
		t.Error("no error writing to a missing directory")
	}
}

// What clusteradm init refuses with spiffe is refused before anything is applied, with --force too.
func TestValidateSPIFFEBeforeApply(t *testing.T) {
	o, _ := spiffeInitOptions(t, []string{"spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org"})
	o.force = true
	o.output = "json"
	if err := o.validate(context.TODO()); err == nil || !strings.Contains(err.Error(), "--output json") {
		t.Errorf("json output: %v", err)
	}
	o.output = "text"
	o.singleton = true
	if err := o.validate(context.TODO()); err == nil || !strings.Contains(err.Error(), "--singleton") {
		t.Errorf("singleton: %v", err)
	}
	o.singleton = false
	o.spiffe.autoApprovedIdentities = []string{"spiffe://ocm-hub.example.org/ocm/cluster/*/agent/registration"}
	if err := genericclioptionsclusteradm.HubMutableFeatureGate.Set("ManagedClusterAutoApproval=false"); err != nil {
		t.Fatal(err)
	}
	if err := o.validate(context.TODO()); err == nil || !strings.Contains(err.Error(), "ManagedClusterAutoApproval") {
		t.Errorf("auto-approved identities without the feature gate, with --force: %v", err)
	}
	// With spiffe next to another driver, json output still prints that driver's token.
	o, _ = spiffeInitOptions(t, []string{"csr", "spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org"})
	o.force, o.output = true, "json"
	if err := o.validate(context.TODO()); err != nil {
		t.Errorf("json output with csr and spiffe: %v", err)
	}
}

// The printed join command uses what the hub gets after a values file merge, and asks join to
// wait when init waited.
func TestSPIFFEJoinAfterValuesMerge(t *testing.T) {
	o, out := spiffeInitOptions(t, []string{"spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org", server: "hub.example.com"})
	if err := o.complete(&cobra.Command{}, nil); err != nil {
		t.Fatal(err)
	}
	// What a values file sets.
	c := o.clusterManagerChartConfig.ClusterManager.RegistrationConfiguration.RegistrationDrivers[0].SPIFFE
	c.ServerSPIFFEID = "spiffe://ocm-hub.example.org/hub/server"
	c.AgentIDTemplate = "spiffe://{trustDomain}/k8s/{cluster}/{role}"
	o.wait = true
	if err := o.printSPIFFEJoin(context.TODO(), out, nil); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--hub-spiffe-id spiffe://ocm-hub.example.org/hub/server",
		"--spiffe-agent-id-template 'spiffe://{trustDomain}/k8s/{cluster}/{role}'", "--wait", "prints the SPIFFE IDs to issue",
		"--spiffe-admission-enforcement <ValidatingAdmissionPolicy|ValidatingWebhook>"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output misses %q:\n%s", want, out)
		}
	}
	o.wait = false
	out.Reset()
	if err := o.printSPIFFEJoin(context.TODO(), out, nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "--wait") || !strings.Contains(out.String(), "status.spiffe.agentIDs") {
		t.Errorf("without --wait:\n%s", out)
	}
	// A merged server ID outside the trust domain is an error.
	c.ServerSPIFFEID = "spiffe://other.example.org/hub"
	if err := o.printSPIFFEJoin(context.TODO(), out, nil); err == nil {
		t.Error("no error for a server ID outside the trust domain")
	}
	o.clusterManagerChartConfig.ClusterManager.RegistrationConfiguration.RegistrationDrivers = nil
	if err := o.printSPIFFEJoin(context.TODO(), out, nil); err == nil {
		t.Error("no error without a spiffe driver")
	}
}

func TestInitSPIFFEFlags(t *testing.T) {
	o, _ := spiffeInitOptions(t, nil, spiffeOptions{})
	cmd := NewCmd(o.ClusteradmFlags, o.Streams)
	for _, f := range []string{"spiffe-trust-domain", "spiffe-agent-trust-domains", "spiffe-server-id", "spiffe-endpoint-type",
		"spiffe-server", "spiffe-agent-id-template", "spiffe-addon-agent-id-template", "spiffe-auto-approved-identities"} {
		if cmd.Flags().Lookup(f) == nil {
			t.Errorf("no --%s flag", f)
		}
	}
}

func TestValidateSPIFFEFlagsWithDriver(t *testing.T) {
	o, _ := spiffeInitOptions(t, []string{"spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org"})
	if err := o.validateSPIFFEFlags(); err != nil {
		t.Errorf("spiffe flags with the spiffe driver: %v", err)
	}
}

func TestSPIFFEAutoApprovedIdentities(t *testing.T) {
	const id = "spiffe://ocm-hub.example.org/ocm/cluster/*/agent/registration"
	setGate := func(on bool) {
		if err := genericclioptionsclusteradm.HubMutableFeatureGate.Set(fmt.Sprintf("ManagedClusterAutoApproval=%v", on)); err != nil {
			t.Fatal(err)
		}
	}
	defer setGate(false)

	setGate(false)
	o, _ := spiffeInitOptions(t, []string{"spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org", autoApprovedIdentities: []string{id}})
	if err := o.validateSPIFFEFlags(); err == nil || !strings.Contains(err.Error(), "ManagedClusterAutoApproval") {
		t.Errorf("without the feature gate: %v", err)
	}

	setGate(true)
	if err := o.validateSPIFFEFlags(); err != nil {
		t.Errorf("with the feature gate: %v", err)
	}
	if err := o.complete(&cobra.Command{}, nil); err != nil {
		t.Fatal(err)
	}
	got := o.clusterManagerChartConfig.ClusterManager.RegistrationConfiguration.RegistrationDrivers[0].SPIFFE.AutoApprovedIdentities
	if len(got) != 1 || got[0] != id {
		t.Errorf("autoApprovedIdentities %v", got)
	}

	o, _ = spiffeInitOptions(t, []string{"csr"}, spiffeOptions{autoApprovedIdentities: []string{id}})
	if err := o.validateSPIFFEFlags(); err == nil || !strings.Contains(err.Error(), "--spiffe-*") {
		t.Errorf("without the spiffe driver: %v", err)
	}

	for _, bad := range []string{"spiffe://ocm-hub.example.org/ocm/clu*ster", "spiffe://ocm-hub.example.org/ocm/../*/agent/registration",
		"spiffe://ocm-hub.example.org/./*"} {
		o, _ = spiffeInitOptions(t, []string{"spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org",
			autoApprovedIdentities: []string{bad}})
		if err := o.complete(&cobra.Command{}, nil); err == nil || !strings.Contains(err.Error(), "--spiffe-auto-approved-identities") {
			t.Errorf("%s: %v", bad, err)
		}
	}
}
