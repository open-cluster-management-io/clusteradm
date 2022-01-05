// Copyright Contributors to the Open Cluster Management project
package addons

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"open-cluster-management.io/clusteradm/pkg/cmd/install/addons/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"
)

const appMgrAddonName = "application-manager"

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("addon options:", "dry-run", o.ClusteradmFlags.DryRun, "names", o.names, "output-file", o.outputFile)

	return nil
}

func (o *Options) validate() error {
	if o.names == "" {
		return fmt.Errorf("names is missing")
	}

	names := strings.Split(o.names, ",")
	for _, n := range names {
		if n != appMgrAddonName {
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
	o.values.addons = addons

	klog.V(3).InfoS("values:", "addon", o.values.addons)

	kubeClient, apiExtensionsClient, dynamicClient, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	return o.runWithClient(kubeClient, apiExtensionsClient, dynamicClient, o.ClusteradmFlags.DryRun)
}

func (o *Options) runWithClient(kubeClient kubernetes.Interface,
	apiExtensionsClient apiextensionsclient.Interface,
	dynamicClient dynamic.Interface,
	dryRun bool) error {

	output := make([]string, 0)
	reader := scenario.GetScenarioResourcesReader()

	applierBuilder := &apply.ApplierBuilder{}
	applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()

	for _, addon := range o.values.addons {
		if addon == appMgrAddonName {
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
				"addon/appmgr/service.yaml",
			}

			out, err := applier.ApplyDirectly(reader, o.values, dryRun, "", files...)
			if err != nil {
				return err
			}
			output = append(output, out...)

			deployments := []string{
				"addon/appmgr/deployment_channel.yaml",
				"addon/appmgr/deployment_subscription.yaml",
				"addon/appmgr/deployment_placementrule.yaml",
				"addon/appmgr/deployment_appsubsummary.yaml",
			}

			out, err = applier.ApplyDeployments(reader, o.values, dryRun, "", deployments...)
			if err != nil {
				return err
			}
			output = append(output, out...)

			fmt.Printf("Installing built-in %s add-on to the Hub cluster...\n", appMgrAddonName)
		}
	}

	return apply.WriteOutput(o.outputFile, output)
}
