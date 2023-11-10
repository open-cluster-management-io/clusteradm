// Copyright Contributors to the Open Cluster Management project
package clustermanager

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	clusteradminit "open-cluster-management.io/clusteradm/pkg/cmd/init"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	init_scenario "open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/version"
	"open-cluster-management.io/clusteradm/pkg/helpers/wait"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("init options:", "dry-run", o.ClusteradmFlags.DryRun)

	f := o.ClusteradmFlags.KubectlFactory
	o.builder = f.NewBuilder()

	restConfig, err := f.ToRESTConfig()
	if err != nil {
		return err
	}

	operatorClient, err := operatorclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	cm, err := operatorClient.OperatorV1().ClusterManagers().Get(context.TODO(), "cluster-manager", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return fmt.Errorf("clustermanager is not installed")
	}
	if err != nil {
		return err
	}

	versionBundle, err := version.GetVersionBundle(o.bundleVersion)
	if err != nil {
		klog.Errorf("unable to retrieve version ", err)
		return err
	}

	o.values = clusteradminit.Values{
		Hub: clusteradminit.Hub{
			Registry: o.registry,
		},
		// reconstruct values from the cluster manager CR.
		RegistrationFeatures: cm.Spec.RegistrationConfiguration.FeatureGates,
		WorkFeatures:         cm.Spec.WorkConfiguration.FeatureGates,
		AddonFeatures:        cm.Spec.AddOnManagerConfiguration.FeatureGates,
		BundleVersion: clusteradminit.BundleVersion{
			RegistrationImageVersion: versionBundle.Registration,
			PlacementImageVersion:    versionBundle.Placement,
			WorkImageVersion:         versionBundle.Work,
			OperatorImageVersion:     versionBundle.Operator,
		},
	}

	if len(cm.Spec.RegistrationConfiguration.AutoApproveUsers) > 0 {
		o.values.AutoApprove = true
	}

	return nil
}

func (o *Options) validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
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
