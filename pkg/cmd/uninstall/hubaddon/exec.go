// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	addonclientset "open-cluster-management.io/api/client/addon/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"open-cluster-management.io/clusteradm/pkg/cmd/install/hubaddon/scenario"
)

const (
	appMgrAddonName          = "application-manager"
	policyFrameworkAddonName = "governance-policy-framework"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("addon options:", "dry-run", o.ClusteradmFlags.DryRun, "names", o.names)
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

	r := reader.NewResourceReader(o.ClusteradmFlags.KubectlFactory, o.ClusteradmFlags.DryRun, o.Streams)

	for _, addon := range o.values.hubAddons {
		if err := o.checkExistingAddon(addon); err != nil {
			return err
		}
		switch addon {
		// Install the Application Management Addon
		case appMgrAddonName:
			err := r.Delete(scenario.Files, o.values, scenario.AppManagerConfigFiles...)
			if err != nil {
				return err
			}

			err = r.Delete(scenario.Files, o.values, scenario.AppManagerDeploymentFiles...)
			if err != nil {
				return err
			}

			fmt.Fprintf(o.Streams.Out, "Uninstalling built-in %s add-on from the Hub cluster...\n", appMgrAddonName)

		// Install the Policy Framework Addon
		case policyFrameworkAddonName:
			err := r.Delete(scenario.Files, o.values, scenario.PolicyFrameworkConfigFiles...)
			if err != nil {
				return fmt.Errorf("Error deploying framework deployment dependencies: %w", err)
			}

			err = r.Delete(scenario.Files, o.values, scenario.PolicyFrameworkDeploymentFiles...)
			if err != nil {
				return fmt.Errorf("Error deploying framework deployments: %w", err)
			}

			fmt.Fprintf(o.Streams.Out, "Uninstalling built-in %s add-on from the Hub cluster...\n", policyFrameworkAddonName)
		}
	}

	return nil
}

func (o *Options) checkExistingAddon(name string) error {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}

	addonClient, err := addonclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	addons, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return err
	}

	if len(addons.Items) > 0 {
		var enabledClusters []string
		for _, addon := range addons.Items {
			enabledClusters = append(enabledClusters, addon.Namespace)
		}
		return fmt.Errorf("there are still addons for %s enabled on some clusters, run `cluster addon disable --names %s "+
			"--clusters %s` to disable addons", name, name, strings.Join(enabledClusters, ","))
	}
	return nil
}
