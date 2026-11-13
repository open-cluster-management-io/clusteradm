// Copyright Contributors to the Open Cluster Management project
package init

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	kubefake "k8s.io/client-go/kubernetes/fake"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

func spiffeInitOptions(t *testing.T, drivers []string, s spiffeOptions) (*Options, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	f := cmdutil.NewFactory(genericclioptions.NewConfigFlags(false))
	o := newOptions(genericclioptionsclusteradm.NewClusteradmFlags(f), genericiooptions.IOStreams{Out: out, ErrOut: out})
	o.registry = "quay.io/open-cluster-management"
	o.bundleVersion = "default"
	o.registrationDrivers = drivers
	if s.endpointType == "" {
		s.endpointType = "loadBalancer"
	}
	o.spiffe = s
	return o, out
}

func TestSPIFFEInitClusterManager(t *testing.T) {
	o, _ := spiffeInitOptions(t, []string{"spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org"})
	if err := o.complete(&cobra.Command{}, nil); err != nil {
		t.Fatal(err)
	}
	cm := o.clusterManagerChartConfig.ClusterManager
	drivers := cm.RegistrationConfiguration.RegistrationDrivers
	if len(drivers) != 1 || drivers[0].AuthType != "spiffe" || drivers[0].SPIFFE == nil ||
		drivers[0].SPIFFE.TrustDomain != "ocm-hub.example.org" {
		t.Errorf("drivers %+v", drivers)
	}
	for name, gates := range map[string][]operatorv1.FeatureGate{
		"registration": cm.RegistrationConfiguration.FeatureGates, "addOnManager": cm.AddOnManagerConfiguration.FeatureGates} {
		found := false
		for _, g := range gates {
			found = found || (g.Feature == "SPIFFERegistration" && g.Mode == operatorv1.FeatureGateModeTypeEnable)
		}
		if !found {
			t.Errorf("%s feature gates %+v miss SPIFFERegistration", name, gates)
		}
	}
	eps := cm.ServerConfiguration.EndpointsExposure
	if len(eps) != 1 || eps[0].Protocol != "grpc-spiffe" || eps[0].GRPC.Type != operatorv1.EndpointTypeLoadBalancer {
		t.Errorf("endpoints %+v", eps)
	}
	if o.clusterManagerChartConfig.CreateBootstrapToken || o.clusterManagerChartConfig.CreateBootstrapSA {
		t.Error("a spiffe-only hub got a bootstrap token or ServiceAccount")
	}

	// The rendered chart has no bootstrap token Secret or ServiceAccount.
	_, raw, err := chart.RenderClusterManagerChart(context.TODO(), o.clusterManagerChartConfig, "open-cluster-management")
	if err != nil {
		t.Fatal(err)
	}
	all := string(bytes.Join(raw, []byte("\n---\n")))
	for _, unwanted := range []string{"bootstrap-token-ocmhub", "agent-registration-bootstrap"} {
		if strings.Contains(all, unwanted) {
			t.Errorf("rendered hub has %s", unwanted)
		}
	}
	if !strings.Contains(all, "trustDomain: ocm-hub.example.org") || !strings.Contains(all, "protocol: grpc-spiffe") {
		t.Errorf("rendered ClusterManager misses the spiffe settings:\n%s", all)
	}
}

func TestSPIFFEInitMixedDrivers(t *testing.T) {
	o, _ := spiffeInitOptions(t, []string{"csr", "grpc", "spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org"})
	o.grpcEndpointType = "hostname"
	o.grpcServer = "grpc.hub.example.com"
	if err := o.complete(&cobra.Command{}, nil); err != nil {
		t.Fatal(err)
	}
	eps := o.clusterManagerChartConfig.ClusterManager.ServerConfiguration.EndpointsExposure
	if len(eps) != 2 || eps[0].Protocol != "grpc" || eps[1].Protocol != "grpc-spiffe" {
		t.Errorf("endpoints %+v, want grpc and grpc-spiffe", eps)
	}
	if o.spiffeOnly() {
		t.Error("spiffeOnly with csr and grpc")
	}
}

func TestSPIFFEInitErrors(t *testing.T) {
	cases := []struct {
		name    string
		drivers []string
		s       spiffeOptions
		want    string
	}{
		{"no trust domain", []string{"spiffe"}, spiffeOptions{}, "--spiffe-trust-domain is required with spiffe"},
		{"server ID outside the trust domain", []string{"spiffe"},
			spiffeOptions{trustDomain: "ocm-hub.example.org", serverID: "spiffe://other.org/hub"}, "not in trust domain"},
		{"typo in a template", []string{"spiffe"},
			spiffeOptions{trustDomain: "ocm-hub.example.org", agentIDTemplate: "spiffe://{trustDomain}/ocm/{clutser}/{role}"}, "clutser"},
		{"hostname without server", []string{"spiffe"},
			spiffeOptions{trustDomain: "ocm-hub.example.org", endpointType: "hostname"}, "--spiffe-server is required"},
		{"unknown endpoint type", []string{"spiffe"},
			spiffeOptions{trustDomain: "ocm-hub.example.org", endpointType: "nodePort"}, "unknown --spiffe-endpoint-type"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o, _ := spiffeInitOptions(t, c.drivers, c.s)
			err := o.complete(&cobra.Command{}, nil)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %v, want %q", err, c.want)
			}
		})
	}

	o, _ := spiffeInitOptions(t, []string{"csr"}, spiffeOptions{trustDomain: "ocm-hub.example.org"})
	if err := o.validateSPIFFEFlags(); err == nil {
		t.Error("spiffe flags accepted without the spiffe driver")
	}
}

func TestSPIFFEInitTemplates(t *testing.T) {
	tpl := "spiffe://{trustDomain}/k8s/{cluster}/ns/{namespace}/ocm/{role}"
	o, out := spiffeInitOptions(t, []string{"spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org", agentIDTemplate: tpl})
	if err := o.complete(&cobra.Command{}, nil); err != nil {
		t.Fatal(err)
	}
	if got := o.clusterManagerChartConfig.ClusterManager.RegistrationConfiguration.RegistrationDrivers[0].SPIFFE.AgentIDTemplate; got != tpl {
		t.Errorf("agentIDTemplate %q", got)
	}
	out.Reset()
	if err := o.printSPIFFEJoin(context.TODO(), out, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "--spiffe-agent-id-template '"+tpl+"'") {
		t.Errorf("join command doesn't pass the template:\n%s", out)
	}
}

func TestSPIFFEJoinCommand(t *testing.T) {
	o, out := spiffeInitOptions(t, []string{"spiffe"}, spiffeOptions{trustDomain: "ocm-hub.example.org"})

	// No address yet: a placeholder and how to read it.
	if err := o.printSPIFFEJoin(context.TODO(), out, kubefake.NewSimpleClientset()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--registration-auth spiffe", "--hub-grpc-endpoint <grpc-spiffe address>",
		"--hub-spiffe-id spiffe://ocm-hub.example.org/ocm/hub/grpc-server", "kubectl -n open-cluster-management-hub get service"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output misses %q:\n%s", want, out)
		}
	}
	if strings.Contains(out.String(), "--hub-token") {
		t.Errorf("printed a token:\n%s", out)
	}

	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: spiffeServiceName, Namespace: spiffeServiceNamespace},
		Spec:   corev1.ServiceSpec{Ports: []corev1.ServicePort{{Name: "grpc", Port: 443}}},
		Status: corev1.ServiceStatus{LoadBalancer: corev1.LoadBalancerStatus{Ingress: []corev1.LoadBalancerIngress{{IP: "10.0.0.7"}}}}}
	out.Reset()
	if err := o.printSPIFFEJoin(context.TODO(), out, kubefake.NewSimpleClientset(svc)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "--hub-grpc-endpoint 10.0.0.7:443") || strings.Contains(out.String(), "kubectl") {
		t.Errorf("output:\n%s", out)
	}
}
