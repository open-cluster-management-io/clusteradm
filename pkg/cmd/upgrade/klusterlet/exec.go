// Copyright Contributors to the Open Cluster Management project
package klusterlet

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stolostron/applier/pkg/apply"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	join_scenario "open-cluster-management.io/clusteradm/pkg/cmd/join/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	version "open-cluster-management.io/clusteradm/pkg/helpers/version"
	"open-cluster-management.io/clusteradm/pkg/helpers/wait"
)

//TODO add to a common folder
//nolint:deadcode,varcheck
const (
	klusterletName                 = "klusterlet"
	registrationOperatorNamespace  = "open-cluster-management"
	klusterletCRD                  = "klusterlets.operator.open-cluster-management.io"
	componentNameRegistrationAgent = "klusterlet-registration-agent"
	componentNameWorkAgent         = "klusterlet-work-agent"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	err = o.ClusteradmFlags.ValidateManagedCluster()
	if err != nil {
		return err
	}

	cfg, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}

	operatorClient, err := operatorclient.NewForConfig(cfg)
	if err != nil {
		return err
	}

	k, err := operatorClient.OperatorV1().Klusterlets().Get(context.TODO(), klusterletName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	klog.V(1).InfoS("init options:", "dry-run", o.ClusteradmFlags.DryRun)
	o.values = Values{
		ClusterName: k.ClusterName,
		Hub: Hub{
			Registry: o.registry,
		},
	}

	versionBundle, err := version.GetVersionBundle(o.bundleVersion)

	if err != nil {
		klog.Errorf("unable to retrieve version ", err)
		return err
	}

	o.values.BundleVersion = BundleVersion{
		RegistrationImageVersion: versionBundle.Registration,
		PlacementImageVersion:    versionBundle.Placement,
		WorkImageVersion:         versionBundle.Work,
		OperatorImageVersion:     versionBundle.Operator,
	}

	return nil
}

func (o *Options) validate() error {

	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	apiExtensionsClient, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	installed, err := helpers.IsKlusterletsInstalled(apiExtensionsClient)
	if err != nil {
		return err
	}

	if !installed {
		return fmt.Errorf("klusterlet is not installed")
	}
	fmt.Fprint(o.Streams.Out, "Klusterlet installed. starting upgrade ")
	fmt.Fprint(o.Streams.Out, "Klusterlet installed. starting upgrade\n")

	return nil
}

func (o *Options) run() error {
	output := make([]string, 0)
	join_reader := join_scenario.GetScenarioResourcesReader()

	kubeClient, apiExtensionsClient, dynamicClient, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	applierBuilder := apply.NewApplierBuilder()
	applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()

	files := []string{
		"join/namespace_agent.yaml",
		"join/namespace_addon.yaml",
		"join/namespace.yaml",
		"join/cluster_role.yaml",
		"join/cluster_role_binding.yaml",
		"join/klusterlets.crd.yaml",
		"join/service_account.yaml",
	}

	out, err := applier.ApplyDirectly(join_reader, o.values, o.ClusteradmFlags.DryRun, "", files...)
	if err != nil {
		return err
	}
	output = append(output, out...)

	out, err = applier.ApplyDeployments(join_reader, o.values, o.ClusteradmFlags.DryRun, "", "join/operator.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	if !o.ClusteradmFlags.DryRun {
		if err := wait.WaitUntilCRDReady(apiExtensionsClient, "clustermanagers.operator.open-cluster-management.io", o.wait); err != nil {
			return err
		}
	}
	if o.wait && !o.ClusteradmFlags.DryRun {
		if err := wait.WaitUntilRegistrationOperatorReady(
			o.ClusteradmFlags.KubectlFactory,
			int64(o.ClusteradmFlags.Timeout)); err != nil {
			return err
		}
	}

	out, err = applier.ApplyCustomResources(join_reader, o.values, o.ClusteradmFlags.DryRun, "", "join/klusterlets.cr.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	fmt.Fprint(o.Streams.Out, "upgraded completed successfully\n")

	return apply.WriteOutput("", output)
}
