// Copyright Contributors to the Open Cluster Management project
package init

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"

	ocmfeature "open-cluster-management.io/api/feature"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	grpcauthn "open-cluster-management.io/sdk-go/pkg/server/grpc/authn"

	"open-cluster-management.io/clusteradm/pkg/helpers"
)

const (
	// spiffeServiceNamespace and spiffeServiceName name the Service of the hub's SPIFFE listener,
	// as the cluster-manager operator renders it.
	spiffeServiceNamespace = "open-cluster-management-hub"
	spiffeServiceName      = "cluster-manager-grpc-server-spiffe"
)

// spiffeOptions are the init flags for the spiffe registration driver.
type spiffeOptions struct {
	trustDomain          string
	agentTrustDomains    []string
	serverID             string
	endpointType         string
	server               string
	agentIDTemplate      string
	addOnAgentIDTemplate string
}

func (o *Options) spiffeEnabled() bool {
	return sets.New(o.registrationDrivers...).Has(operatorv1.SPIFFEAuthType)
}

// spiffeOnly reports whether spiffe is the hub's only registration driver. Then init creates no
// bootstrap token or ServiceAccount: agents need none.
func (o *Options) spiffeOnly() bool {
	return o.spiffeEnabled() && len(sets.New(o.registrationDrivers...)) == 1
}

// spiffeConfig builds the ClusterManager's spiffe driver settings and checks them as the operator
// will, so a typo fails before anything is applied.
func (o *Options) spiffeConfig() (*operatorv1.SPIFFERegistrationConfig, string, error) {
	if o.spiffe.trustDomain == "" {
		return nil, "", fmt.Errorf("--spiffe-trust-domain is required with spiffe")
	}
	c := &operatorv1.SPIFFERegistrationConfig{
		TrustDomain:          o.spiffe.trustDomain,
		AgentTrustDomains:    o.spiffe.agentTrustDomains,
		ServerSPIFFEID:       o.spiffe.serverID,
		AgentIDTemplate:      o.spiffe.agentIDTemplate,
		AddOnAgentIDTemplate: o.spiffe.addOnAgentIDTemplate,
	}
	serverID, err := spiffeServerID(c)
	if err != nil {
		return nil, "", err
	}
	return c, serverID, nil
}

// spiffeServerID checks the settings as the cluster-manager operator does: the server ID is in
// the trust domain, the ID templates are valid and no agent ID can equal the server's.
func spiffeServerID(c *operatorv1.SPIFFERegistrationConfig) (string, error) {
	td, err := spiffeid.TrustDomainFromString(c.TrustDomain)
	if err != nil {
		return "", fmt.Errorf("--spiffe-trust-domain %q: %w", c.TrustDomain, err)
	}
	serverID := c.ServerSPIFFEID
	if serverID == "" {
		serverID = "spiffe://" + td.Name() + "/ocm/hub/grpc-server"
	}
	sid, err := spiffeid.FromString(serverID)
	if err != nil {
		return "", fmt.Errorf("--spiffe-server-id %q: %w", serverID, err)
	}
	if !sid.MemberOf(td) {
		return "", fmt.Errorf("--spiffe-server-id %s is not in trust domain %s", sid, td.Name())
	}
	scheme, err := grpcauthn.NewIDScheme(c.GetAgentIDTemplate(), c.GetAddOnAgentIDTemplate(), c.GetAgentTrustDomains())
	if err != nil {
		return "", err
	}
	if err := scheme.CheckServerID(sid); err != nil {
		return "", err
	}
	return sid.String(), nil
}

// spiffeEndpoint is how the SPIFFE listener is exposed.
func (o *Options) spiffeEndpoint() (operatorv1.EndpointExposure, error) {
	endpoint := operatorv1.EndpointExposure{Protocol: operatorv1.GRPCSPIFFEProtocol}
	switch operatorv1.EndpointExposureType(o.spiffe.endpointType) {
	case operatorv1.EndpointTypeHostname:
		if o.spiffe.server == "" {
			return endpoint, fmt.Errorf("--spiffe-server is required with --spiffe-endpoint-type hostname")
		}
		endpoint.GRPC = &operatorv1.Endpoint{Type: operatorv1.EndpointTypeHostname,
			Hostname: &operatorv1.HostnameConfig{Host: o.spiffe.server}}
	case operatorv1.EndpointTypeLoadBalancer:
		endpoint.GRPC = &operatorv1.Endpoint{Type: operatorv1.EndpointTypeLoadBalancer,
			LoadBalancer: &operatorv1.LoadBalancerConfig{Host: o.spiffe.server}}
	default:
		return endpoint, fmt.Errorf("unknown --spiffe-endpoint-type %q; use hostname or loadBalancer", o.spiffe.endpointType)
	}
	return endpoint, nil
}

// validateSPIFFEFlags refuses spiffe flags without the spiffe driver.
func (o *Options) validateSPIFFEFlags() error {
	if o.spiffeEnabled() {
		return nil
	}
	s := o.spiffe
	if s.trustDomain != "" || len(s.agentTrustDomains) > 0 || s.serverID != "" || s.server != "" ||
		s.agentIDTemplate != "" || s.addOnAgentIDTemplate != "" {
		return fmt.Errorf("the --spiffe-* flags need --registration-drivers to include spiffe")
	}
	return nil
}

// enableFeature returns gates with feature enabled.
func enableFeature(gates []operatorv1.FeatureGate, feature string) []operatorv1.FeatureGate {
	for i := range gates {
		if gates[i].Feature == feature {
			gates[i].Mode = operatorv1.FeatureGateModeTypeEnable
			return gates
		}
	}
	return append(gates, operatorv1.FeatureGate{Feature: feature, Mode: operatorv1.FeatureGateModeTypeEnable})
}

// applySPIFFE adds the spiffe driver's feature gates and endpoint to the ClusterManager.
func (o *Options) applySPIFFE() error {
	if !o.spiffeEnabled() {
		return nil
	}
	endpoint, err := o.spiffeEndpoint()
	if err != nil {
		return err
	}
	cm := &o.clusterManagerChartConfig.ClusterManager
	gate := string(ocmfeature.SPIFFERegistration)
	cm.RegistrationConfiguration.FeatureGates = enableFeature(cm.RegistrationConfiguration.FeatureGates, gate)
	cm.AddOnManagerConfiguration.FeatureGates = enableFeature(cm.AddOnManagerConfiguration.FeatureGates, gate)
	cm.ServerConfiguration.EndpointsExposure = append(cm.ServerConfiguration.EndpointsExposure, endpoint)
	return nil
}

// spiffeHubEndpoint returns host:443 of the SPIFFE listener, or "" when it isn't known yet.
func (o *Options) spiffeHubEndpoint(ctx context.Context, kubeClient kubernetes.Interface) string {
	if o.spiffe.server != "" {
		return net.JoinHostPort(o.spiffe.server, "443")
	}
	if kubeClient == nil || o.ClusteradmFlags.DryRun {
		return ""
	}
	svc, err := kubeClient.CoreV1().Services(spiffeServiceNamespace).Get(ctx, spiffeServiceName, metav1.GetOptions{})
	if err != nil {
		return ""
	}
	return serviceAddress(svc)
}

func serviceAddress(svc *corev1.Service) string {
	port := "443"
	for _, p := range svc.Spec.Ports {
		if p.Name == "grpc" || p.Port == 443 {
			port = strconv.Itoa(int(p.Port))
			break
		}
	}
	for _, ing := range svc.Status.LoadBalancer.Ingress {
		if ing.Hostname != "" {
			return net.JoinHostPort(ing.Hostname, port)
		}
		if ing.IP != "" {
			return net.JoinHostPort(ing.IP, port)
		}
	}
	return ""
}

// printSPIFFEJoin prints the join command for spiffe clusters. It has no token.
func (o *Options) printSPIFFEJoin(ctx context.Context, w io.Writer, kubeClient kubernetes.Interface) error {
	_, serverID, err := o.spiffeConfig()
	if err != nil {
		return err
	}
	endpoint := o.spiffeHubEndpoint(ctx, kubeClient)
	known := endpoint != ""
	if !known {
		endpoint = "<grpc-spiffe address>"
	}
	fmt.Fprintf(w, "Your hub is ready for SPIFFE registration. On each managed cluster, after its SPIRE entries exist, run:\n\n"+
		"    %s join --cluster-name <cluster_name> --registration-auth spiffe \\\n"+
		"      --hub-grpc-endpoint %s \\\n"+
		"      --hub-spiffe-id %s",
		helpers.GetExampleHeader(), endpoint, serverID)
	if o.spiffe.agentIDTemplate != "" {
		fmt.Fprintf(w, " \\\n      --spiffe-agent-id-template '%s'", o.spiffe.agentIDTemplate)
	}
	if o.spiffe.addOnAgentIDTemplate != "" {
		fmt.Fprintf(w, " \\\n      --spiffe-addon-agent-id-template '%s'", o.spiffe.addOnAgentIDTemplate)
	}
	fmt.Fprint(w, "\n\n")
	if !known {
		fmt.Fprintf(w, "The SPIFFE listener has no address yet. Read it once its load balancer is ready:\n\n"+
			"    kubectl -n %s get service %s -o jsonpath='{.status.loadBalancer.ingress[0]}'\n\n",
			spiffeServiceNamespace, spiffeServiceName)
	}
	return nil
}

// finishSPIFFEOnly writes the outputs of a hub whose only driver is spiffe: no token is read or
// printed.
func (o *Options) finishSPIFFEOnly(ctx context.Context, r rawApplied, kubeClient kubernetes.Interface) error {
	if len(o.outputFile) > 0 {
		if err := writeFile(o.outputFile, r.RawAppliedResources()); err != nil {
			return err
		}
	}
	if o.output == "json" {
		return fmt.Errorf("--output json prints a join token; spiffe needs none, use --output text")
	}
	fmt.Fprint(o.Streams.Out, "The multicluster hub control plane has been initialized successfully!\n\n")
	return o.printSPIFFEJoin(ctx, o.Streams.Out, kubeClient)
}

type rawApplied interface{ RawAppliedResources() []byte }

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
