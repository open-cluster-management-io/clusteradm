// Copyright Contributors to the Open Cluster Management project
package clustermanager

import (
	"fmt"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"

	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/klog/v2"
	init_scenario "open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/version"
	"open-cluster-management.io/clusteradm/pkg/helpers/wait"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("init options:", "dry-run", o.ClusteradmFlags.DryRun)
	o.values = Values{
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

	f := o.ClusteradmFlags.KubectlFactory
	o.builder = f.NewBuilder()

	return nil
}

func (o *Options) validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}

	apiExtensionsClient, err := apiextensionsclient.NewForConfig(restConfig)
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

	fmt.Fprint(o.Streams.Out, "clustermanager installed. starting upgrade\n")

	return nil
}

func (o *Options) run() error {
	r := reader.NewResourceReader(o.builder, o.ClusteradmFlags.DryRun, o.Streams)

	_, apiExtensionsClient, _, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	files := []string{
		"init/clustermanager_cluster_role.yaml",
		"init/clustermanager_cluster_role_binding.yaml",
		"init/clustermanagers.crd.yaml",
		"init/clustermanager_sa.yaml",
	}

	err = r.Apply(init_scenario.Files, o.values, files...)
	if err != nil {
		return err
	}

	err = r.Apply(init_scenario.Files, o.values, "init/operator.yaml")
	if err != nil {
		return err
	}

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

	err = r.Apply(init_scenario.Files, o.values, "init/clustermanager.cr.yaml")
	if err != nil {
		return err
	}

	fmt.Fprint(o.Streams.Out, "upgraded completed successfully\n")
	return nil
}
