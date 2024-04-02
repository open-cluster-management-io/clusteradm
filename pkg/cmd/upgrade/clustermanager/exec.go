// Copyright Contributors to the Open Cluster Management project
package clustermanager

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	init_scenario "open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/wait"
	"open-cluster-management.io/clusteradm/pkg/version"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("init options:", "dry-run", o.ClusteradmFlags.DryRun)

	f := o.ClusteradmFlags.KubectlFactory
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
		klog.Errorf("unable to retrieve version: %v", err)
		return err
	}

	o.values = init_scenario.Values{
		Hub: init_scenario.Hub{
			Registry: o.registry,
		},
		BundleVersion: init_scenario.BundleVersion{
			RegistrationImageVersion: versionBundle.Registration,
			PlacementImageVersion:    versionBundle.Placement,
			WorkImageVersion:         versionBundle.Work,
			AddonManagerImageVersion: versionBundle.AddonManager,
			OperatorImageVersion:     versionBundle.Operator,
		},
	}

	// reconstruct values from the cluster manager CR.
	if cm.Spec.RegistrationConfiguration != nil {
		o.values.RegistrationFeatures = cm.Spec.RegistrationConfiguration.FeatureGates
		if len(cm.Spec.RegistrationConfiguration.AutoApproveUsers) > 0 {
			o.values.AutoApprove = true
		}
	}
	if cm.Spec.WorkConfiguration != nil {
		o.values.WorkFeatures = cm.Spec.WorkConfiguration.FeatureGates
	}
	if cm.Spec.AddOnManagerConfiguration != nil {
		o.values.AddonFeatures = cm.Spec.AddOnManagerConfiguration.FeatureGates
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
	r := reader.NewResourceReader(o.ClusteradmFlags.KubectlFactory, o.ClusteradmFlags.DryRun, o.Streams)

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
