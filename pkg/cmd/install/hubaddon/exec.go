// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"fmt"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"open-cluster-management.io/clusteradm/pkg/cmd/install/hubaddon/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers/version"
)

const (
	appMgrAddonName          = "application-manager"
	policyFrameworkAddonName = "governance-policy-framework"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("addon options:", "dry-run", o.ClusteradmFlags.DryRun, "names", o.names, "output-file", o.outputFile)
	f := o.ClusteradmFlags.KubectlFactory
	o.builder = f.NewBuilder()
	return nil
}

func (o *Options) validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	if o.names == "" {
		return fmt.Errorf("names is missing")
	}

	names := strings.Split(o.names, ",")
	for _, n := range names {
		switch n {
		case appMgrAddonName:
			continue
		case policyFrameworkAddonName:
			continue
		default:
			return fmt.Errorf("invalid add-on name %s", n)
		}
	}

	versionBundle, err := version.GetVersionBundle(o.bundleVersion)

	if err != nil {
		klog.Errorf("unable to retrieve version "+o.bundleVersion, err)
		return err
	}

	o.values.BundleVersion = BundleVersion{
		AppAddon:    versionBundle.AppAddon,
		PolicyAddon: versionBundle.PolicyAddon,
	}

	return nil
}

func (o *Options) run() error {
	alreadyProvidedAddons := make(map[string]bool)
	addons := make([]string, 0)
	names := strings.Split(o.names, ",")
	for _, n := range names {
		if _, ok := alreadyProvidedAddons[n]; !ok {
			alreadyProvidedAddons[n] = true
			addons = append(addons, strings.TrimSpace(n))
		}
	}
	o.values.hubAddons = addons

	klog.V(3).InfoS("values:", "addon", o.values.hubAddons)

	return o.runWithClient()
}

func (o *Options) runWithClient() error {

	r := reader.NewResourceReader(o.builder, o.ClusteradmFlags.DryRun, o.Streams)

	for _, addon := range o.values.hubAddons {
		switch addon {
		// Install the Application Management Addon
		case appMgrAddonName:
			files := []string{
				"addon/appmgr/clusterrole_agent.yaml",
				"addon/appmgr/clusterrole_binding.yaml",
				"addon/appmgr/clusterrole.yaml",
				"addon/appmgr/crd_channel.yaml",
				"addon/appmgr/crd_helmrelease.yaml",
				"addon/appmgr/crd_placementrule.yaml",
				"addon/appmgr/crd_subscription.yaml",
				"addon/appmgr/crd_subscriptionstatuses.yaml",
				"addon/appmgr/crd_report.yaml",
				"addon/appmgr/crd_clusterreport.yaml",
				"addon/appmgr/service_account.yaml",
				"addon/appmgr/service_metrics.yaml",
				"addon/appmgr/service_operator.yaml",
			}

			err := r.Apply(scenario.Files, o.values, files...)
			if err != nil {
				return err
			}

			deployments := []string{
				"addon/appmgr/deployment_channel.yaml",
				"addon/appmgr/deployment_subscription.yaml",
				"addon/appmgr/deployment_placementrule.yaml",
				"addon/appmgr/deployment_appsubsummary.yaml",
			}
			err = r.Apply(scenario.Files, o.values, deployments...)
			if err != nil {
				return err
			}

			fmt.Fprintf(o.Streams.Out, "Installing built-in %s add-on to the Hub cluster...\n", appMgrAddonName)

		// Install the Policy Framework Addon
		case policyFrameworkAddonName:
			files := []string{
				"addon/policy/addon-controller_clusterrole.yaml",
				"addon/policy/addon-controller_clusterrolebinding.yaml",
				"addon/policy/addon-controller_role.yaml",
				"addon/policy/addon-controller_rolebinding.yaml",
				"addon/policy/addon-controller_serviceaccount.yaml",
				"addon/policy/policy.open-cluster-management.io_placementbindings.yaml",
				"addon/policy/policy.open-cluster-management.io_policies.yaml",
				"addon/policy/policy.open-cluster-management.io_policyautomations.yaml",
				"addon/policy/policy.open-cluster-management.io_policysets.yaml",
				"addon/policy/propagator_clusterrole.yaml",
				"addon/policy/propagator_clusterrolebinding.yaml",
				"addon/policy/propagator_role.yaml",
				"addon/policy/propagator_rolebinding.yaml",
				"addon/policy/propagator_serviceaccount.yaml",
				"addon/appmgr/crd_placementrule.yaml",
			}

			err := r.Apply(scenario.Files, o.values, files...)
			if err != nil {
				return err
			}

			deployments := []string{
				"addon/policy/addon-controller_deployment.yaml",
				"addon/policy/propagator_deployment.yaml",
			}

			err = r.Apply(scenario.Files, o.values, deployments...)
			if err != nil {
				return err
			}

			fmt.Fprintf(o.Streams.Out, "Installing built-in %s add-on to the Hub cluster...\n", policyFrameworkAddonName)
		}
	}

	if len(o.outputFile) > 0 {
		sh, err := os.OpenFile(o.outputFile, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(sh, "%s", string(r.RawAppliedResources()))
		if err != nil {
			return err
		}
		if err := sh.Close(); err != nil {
			return err
		}
	}

	return nil
}
