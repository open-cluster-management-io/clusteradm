// Copyright Contributors to the Open Cluster Management project
package init

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"

	ocmfeature "open-cluster-management.io/api/feature"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	grpcauthn "open-cluster-management.io/sdk-go/pkg/server/grpc/authn"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
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
	// autoApprovedIdentities are the SPIFFE IDs whose clusters are accepted automatically.
	autoApprovedIdentities []string
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
		TrustDomain:            o.spiffe.trustDomain,
		AgentTrustDomains:      o.spiffe.agentTrustDomains,
		ServerSPIFFEID:         o.spiffe.serverID,
		AgentIDTemplate:        o.spiffe.agentIDTemplate,
		AddOnAgentIDTemplate:   o.spiffe.addOnAgentIDTemplate,
		AutoApprovedIdentities: o.spiffe.autoApprovedIdentities,
	}
	for _, id := range c.AutoApprovedIdentities {
		if !autoApprovedIdentityPattern.MatchString(id) {
			return nil, "", fmt.Errorf("--spiffe-auto-approved-identities %q: want a SPIFFE ID in which * stands for a whole path segment", id)
		}
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

// autoApprovedIdentityPattern is the ClusterManager CRD's pattern for autoApprovedIdentities.
// A segment is * or a name that isn't . or .., which SPIFFE forbids.
var autoApprovedIdentityPattern = regexp.MustCompile(
	`^spiffe://[a-z0-9._-]+(/(([A-Za-z0-9._-]*[A-Za-z0-9_-][A-Za-z0-9._-]*|[.]{3,})|\*))+$`)

// validateSPIFFEFlags refuses spiffe flags without the spiffe driver, and auto-approved
// identities without the ManagedClusterAutoApproval feature gate, as for the other drivers. It
// also refuses, before anything is applied, what the spiffe driver doesn't support: the singleton
// control plane, and --output json for a hub with no join token.
func (o *Options) validateSPIFFEFlags() error {
	if o.spiffeEnabled() && o.singleton {
		return fmt.Errorf("--registration-drivers spiffe isn't supported with --singleton: the singleton control plane doesn't apply the spiffe settings")
	}
	if o.spiffeOnly() && o.output == "json" {
		return fmt.Errorf("--output json prints a join token; spiffe needs none, use --output text")
	}
	if len(o.spiffe.autoApprovedIdentities) > 0 &&
		!genericclioptionsclusteradm.HubMutableFeatureGate.Enabled("ManagedClusterAutoApproval") {
		return fmt.Errorf("should enable feature gate ManagedClusterAutoApproval before passing --spiffe-auto-approved-identities")
	}
	if o.spiffeEnabled() {
		return nil
	}
	s := o.spiffe
	if s.trustDomain != "" || len(s.agentTrustDomains) > 0 || s.serverID != "" || s.server != "" ||
		s.agentIDTemplate != "" || s.addOnAgentIDTemplate != "" || len(s.autoApprovedIdentities) > 0 {
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

// effectiveSPIFFE returns the spiffe driver settings the hub gets, after a values file merge, and
// the hub's SPIFFE ID they give.
func (o *Options) effectiveSPIFFE() (*operatorv1.SPIFFERegistrationConfig, string, error) {
	for _, d := range o.clusterManagerChartConfig.ClusterManager.RegistrationConfiguration.RegistrationDrivers {
		if d.AuthType == operatorv1.SPIFFEAuthType && d.SPIFFE != nil {
			serverID, err := spiffeServerID(d.SPIFFE)
			return d.SPIFFE, serverID, err
		}
	}
	return nil, "", fmt.Errorf("the cluster-manager configuration has no spiffe registration driver")
}

// spiffeJoinCommand is the join command for spiffe clusters, from the settings the hub gets. It
// has no token. clusterName and enforcement are what the command line holds for them.
func (o *Options) spiffeJoinCommand(endpoint, clusterName, enforcement string) (string, error) {
	c, serverID, err := o.effectiveSPIFFE()
	if err != nil {
		return "", err
	}
	cmd := fmt.Sprintf("%s join --cluster-name %s --registration-auth spiffe \\\n"+
		"      --hub-grpc-endpoint %s \\\n"+
		"      --hub-spiffe-id %s \\\n"+
		"      --spiffe-admission-enforcement %s",
		helpers.GetExampleHeader(), clusterName, endpoint, serverID, enforcement)
	if c.AgentIDTemplate != "" {
		cmd += fmt.Sprintf(" \\\n      --spiffe-agent-id-template '%s'", c.AgentIDTemplate)
	}
	if c.AddOnAgentIDTemplate != "" {
		cmd += fmt.Sprintf(" \\\n      --spiffe-addon-agent-id-template '%s'", c.AddOnAgentIDTemplate)
	}
	// As for the token join command: a foreground init gives a foreground join.
	if o.wait {
		cmd += " \\\n      --wait"
	}
	return cmd, nil
}

// printSPIFFEJoin prints the join command for spiffe clusters.
func (o *Options) printSPIFFEJoin(ctx context.Context, w io.Writer, kubeClient kubernetes.Interface) error {
	endpoint := o.spiffeHubEndpoint(ctx, kubeClient)
	known := endpoint != ""
	if !known {
		endpoint = "<grpc-spiffe address>"
	}
	cmd, err := o.spiffeJoinCommand(endpoint, "<cluster_name>", "<ValidatingAdmissionPolicy|ValidatingWebhook>")
	if err != nil {
		return err
	}
	next := "Then read the SPIFFE IDs to issue from the Klusterlet's status.spiffe.agentIDs once it is SPIFFEAdmissionReady."
	if o.wait {
		next = "It waits until the Klusterlet has secured its namespaces (SPIFFEAdmissionReady) and prints the SPIFFE IDs to issue."
	}
	fmt.Fprintf(w, "Your hub is ready for SPIFFE registration. On each managed cluster, run the following.\n%s\n"+
		"ValidatingAdmissionPolicy needs Kubernetes 1.30 or later; ValidatingWebhook works on every version.\n\n    %s\n\n",
		next, cmd)
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
	// The join command as a script that takes the cluster name and the admission enforcement.
	if len(o.outputJoinCommandFile) > 0 {
		endpoint := o.spiffeHubEndpoint(ctx, kubeClient)
		if endpoint == "" {
			endpoint = "<grpc-spiffe address>"
		}
		script, err := o.spiffeJoinCommand(endpoint, "$1", "$2")
		if err != nil {
			return err
		}
		if err := os.WriteFile(o.outputJoinCommandFile, []byte(script+"\n"), 0o755); err != nil {
			return fmt.Errorf("--output-join-command-file: %w", err)
		}
	}
	fmt.Fprint(o.Streams.Out, "The multicluster hub control plane has been initialized successfully!\n\n")
	return o.printSPIFFEJoin(ctx, o.Streams.Out, kubeClient)
}

type rawApplied interface{ RawAppliedResources() []byte }

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
