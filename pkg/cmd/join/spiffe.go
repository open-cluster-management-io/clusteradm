// Copyright Contributors to the Open Cluster Management project
package join

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8swait "k8s.io/apimachinery/pkg/util/wait"

	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	ocmfeature "open-cluster-management.io/api/feature"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/sdk-go/pkg/spiffe/idtemplate"

	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
	"open-cluster-management.io/clusteradm/pkg/helpers/wait"
)

// spiffeOptions are the join flags for the spiffe registration driver.
type spiffeOptions struct {
	hubGRPCEndpoint      string
	hubSPIFFEID          string
	trustDomain          string
	workloadAPISocket    string
	agentIDTemplate      string
	addOnAgentIDTemplate string
}

func (o *Options) isSPIFFE() bool { return o.registrationAuth == operatorv1.SPIFFEAuthType }

// checkSPIFFEFlags checks the join flags of a spiffe cluster. Agents use only their SVIDs, so a
// token or hub API server passed by mistake is refused rather than ignored.
func (o *Options) checkSPIFFEFlags(cmd *cobra.Command) error {
	if o.token != "" {
		return fmt.Errorf("--hub-token is not used with spiffe: agents authenticate with their SVIDs only")
	}
	if o.hubAPIServer != "" || o.caFile != "" || o.grpcServer != "" || o.grpcCAFile != "" {
		return fmt.Errorf("--hub-apiserver, --ca-file, --grpc-server and --grpc-ca-file are not used with spiffe; " +
			"use --hub-grpc-endpoint and --hub-spiffe-id")
	}
	if o.clusterName == "" {
		return fmt.Errorf("--cluster-name is required with spiffe")
	}
	if o.spiffe.hubGRPCEndpoint == "" {
		return fmt.Errorf("--hub-grpc-endpoint is required with spiffe")
	}
	if o.spiffe.hubSPIFFEID == "" {
		return fmt.Errorf("--hub-spiffe-id is required with spiffe")
	}
	if _, err := spiffeid.FromString(o.spiffe.hubSPIFFEID); err != nil {
		return fmt.Errorf("--hub-spiffe-id %q: %w", o.spiffe.hubSPIFFEID, err)
	}
	if o.spiffe.trustDomain != "" {
		if _, err := spiffeid.TrustDomainFromString(o.spiffe.trustDomain); err != nil {
			return fmt.Errorf("--spiffe-trust-domain %q: %w", o.spiffe.trustDomain, err)
		}
	}
	if t := o.spiffe.agentIDTemplate; t != "" {
		if _, err := idtemplate.Compile(idtemplate.KindAgent, t); err != nil {
			return fmt.Errorf("--spiffe-agent-id-template: %w", err)
		}
	}
	if t := o.spiffe.addOnAgentIDTemplate; t != "" {
		if _, err := idtemplate.Compile(idtemplate.KindAddOnAgent, t); err != nil {
			return fmt.Errorf("--spiffe-addon-agent-id-template: %w", err)
		}
	}
	if cmd != nil && cmd.Flags().Changed("addon-kubeclient-registration-auth") &&
		o.addonKubeClientRegistrationAuth != operatorv1.SPIFFEAuthType {
		return fmt.Errorf("with spiffe, add-on agents register with their own SVIDs; "+
			"--addon-kubeclient-registration-auth %s is not supported", o.addonKubeClientRegistrationAuth)
	}
	o.addonKubeClientRegistrationAuth = operatorv1.SPIFFEAuthType
	return nil
}

// setSPIFFEDriver sets the Klusterlet's spiffe registration settings and feature gates.
func (o *Options) setSPIFFEDriver() {
	rc := &o.klusterletChartConfig.Klusterlet.RegistrationConfiguration
	rc.RegistrationDriver = operatorv1.RegistrationDriver{
		AuthType: operatorv1.SPIFFEAuthType,
		SPIFFE: &operatorv1.KlusterletSPIFFEConfig{
			HubEndpoint:          o.spiffe.hubGRPCEndpoint,
			HubSPIFFEID:          o.spiffe.hubSPIFFEID,
			TrustDomain:          o.spiffe.trustDomain,
			WorkloadAPISocket:    o.spiffe.workloadAPISocket,
			AgentIDTemplate:      o.spiffe.agentIDTemplate,
			AddOnAgentIDTemplate: o.spiffe.addOnAgentIDTemplate,
		},
	}
	rc.AddOnKubeClientRegistrationDriver = &operatorv1.AddOnRegistrationDriver{AuthType: operatorv1.SPIFFEAuthType}
	rc.FeatureGates = enableFeature(rc.FeatureGates, string(ocmfeature.SPIFFERegistration))
}

// enableSPIFFEWorkGate enables the work agent's SPIFFE feature gate.
func (o *Options) enableSPIFFEWorkGate() {
	wc := &o.klusterletChartConfig.Klusterlet.WorkConfiguration
	wc.FeatureGates = enableFeature(wc.FeatureGates, string(ocmfeature.SPIFFERegistration))
}

func enableFeature(gates []operatorv1.FeatureGate, feature string) []operatorv1.FeatureGate {
	for i := range gates {
		if gates[i].Feature == feature {
			gates[i].Mode = operatorv1.FeatureGateModeTypeEnable
			return gates
		}
	}
	return append(gates, operatorv1.FeatureGate{Feature: feature, Mode: operatorv1.FeatureGateModeTypeEnable})
}

// waitForSPIFFEAdmission waits until the Klusterlet has secured its namespaces and returns it.
// Only then may the platform create the SPIRE entries for its agents.
func waitForSPIFFEAdmission(ctx context.Context, w io.Writer, client operatorclient.Interface, timeout int64,
	name string) (*operatorv1.Klusterlet, error) {
	phase := &atomic.Value{}
	phase.Store("")
	spinner := printer.NewSpinnerWithStatus(w,
		"Waiting for the Klusterlet to secure its namespaces (SPIFFEAdmissionReady)...",
		500*time.Millisecond, "done.\n", func() string { return phase.Load().(string) })
	spinner.Start()
	defer spinner.Stop()

	var klusterlet *operatorv1.Klusterlet
	err := k8swait.PollUntilContextTimeout(ctx, 3*time.Second, time.Duration(timeout)*time.Second, true,
		func(ctx context.Context) (bool, error) {
			k, err := client.OperatorV1().Klusterlets().Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return wait.HandlePollError(ctx, err, phase)
			}
			if c := meta.FindStatusCondition(k.Status.Conditions, operatorv1.ConditionSPIFFEAdmissionReady); c != nil {
				phase.Store(c.Reason)
			}
			klusterlet = k
			return meta.IsStatusConditionTrue(k.Status.Conditions, operatorv1.ConditionSPIFFEAdmissionReady), nil
		})
	return klusterlet, wait.TimeoutError(err, phase, "timed out waiting for klusterlet %s to become SPIFFEAdmissionReady", name)
}

// printSPIFFENextSteps tells the admin which SPIFFE IDs to issue and how to accept the cluster.
func (o *Options) printSPIFFENextSteps(w io.Writer, klusterlet *operatorv1.Klusterlet) {
	cluster := o.klusterletChartConfig.Klusterlet.ClusterName
	if klusterlet == nil {
		fmt.Fprintf(w, "The Klusterlet %s is applied. Wait until its SPIFFEAdmissionReady condition is True before "+
			"creating SPIRE entries for its agents; their SPIFFE IDs are in status.spiffe.agentIDs:\n\n"+
			"    kubectl get klusterlet %s -o jsonpath='{.status.spiffe.agentIDs}'\n\n",
			o.klusterletChartConfig.Klusterlet.Name, o.klusterletChartConfig.Klusterlet.Name)
	} else {
		fmt.Fprintf(w, "Create SPIRE entries for these SPIFFE IDs for %s now:\n", cluster)
		tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
		if klusterlet.Status.SPIFFE != nil {
			for _, id := range klusterlet.Status.SPIFFE.AgentIDs {
				fmt.Fprintf(tw, "  %s\t%s\n", id.Role, id.SPIFFEID)
			}
		}
		_ = tw.Flush()
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "Then accept it on the hub:\n\n    %s accept --clusters %s\n\n", helpers.GetExampleHeader(), cluster)
}
