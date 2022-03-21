// Copyright Contributors to the Open Cluster Management project
package clustermanager

import (
	"fmt"

	init_scenario "open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"
	"open-cluster-management.io/clusteradm/pkg/helpers/wait"

	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/klog/v2"
	version "open-cluster-management.io/clusteradm/pkg/helpers/version"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("init options:", "dry-run", o.ClusteradmFlags.DryRun, )
	o.values = Values{
		Hub: Hub{			
			Registry:  o.registry,
		},
		}
	

	versionBundle, err := version.GetVersionBundle(o.bundleVersion)

	if err != nil {
		klog.Errorf("unable to retrive version ", err)
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
	apiExtensionsClient , err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	installed, err := helpers.IsClusterManagerInstalled(apiExtensionsClient)
	if err != nil {
		return err
	}

	if !installed {
		return fmt.Errorf("clustermanager is not installed")
	}

	//TODO check desired version is greater then current version 

	fmt.Fprint(o.Streams.Out, "clustermanager installed. starting upgrade")


	return nil
}

func (o *Options) run() error {
	output := make([]string, 0)
    reader := init_scenario.GetScenarioResourcesReader() 

	kubeClient, apiExtensionsClient, dynamicClient, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	applierBuilder := &apply.ApplierBuilder{}
	applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()

	files := []string{
		"init/clustermanager_cluster_role.yaml",
		"init/clustermanager_cluster_role_binding.yaml",
		"init/clustermanagers.crd.yaml",
		"init/clustermanager_sa.yaml", 
	}
	
	out, err := applier.ApplyDirectly(reader, o.values, o.ClusteradmFlags.DryRun, "", files...)
	if err != nil {
		return err
	}
	output = append(output, out...)


	out, err = applier.ApplyDeployments(reader, o.values, o.ClusteradmFlags.DryRun, "", "init/operator.yaml") 
	if err != nil {
		return err
	}
	output = append(output, out...)

	out, err = applier.ApplyDirectly(reader, o.values, o.ClusteradmFlags.DryRun, "", "init/clustermanagers.crd.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	if o.wait && !o.ClusteradmFlags.DryRun {
		if err := wait.WaitUntilCRDReady(apiExtensionsClient,"clustermanagers.operator.open-cluster-management.io" )   ; err != nil {
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

	out, err = applier.ApplyCustomResources(reader, o.values, o.ClusteradmFlags.DryRun, "", "init/clustermanager.cr.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)


	fmt.Fprint(o.Streams.Out,"upgraded completed successfully")
	return apply.WriteOutput("", output)
}


