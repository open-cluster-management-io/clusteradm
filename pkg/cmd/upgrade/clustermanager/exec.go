// Copyright Contributors to the Open Cluster Management project
package clustermanager

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"open-cluster-management.io/clusteradm/pkg/version"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/wait"
)

func (o *Options) complete(_ *cobra.Command, _ []string) (err error) {
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

	bundleVersion, err := version.GetVersionBundle(o.bundleVersion, o.versionBundleFile)
	if err != nil {
		return err
	}

	o.clusterManagerChartConfig.Images = chart.ImagesConfig{
		Registry: o.registry,
		Tag:      bundleVersion.OCM,
	}

	if cm.Spec.ResourceRequirement != nil {
		o.clusterManagerChartConfig.ClusterManager = chart.ClusterManagerConfig{
			ResourceRequirement: *cm.Spec.ResourceRequirement,
		}
	}

	// reconstruct values from the cluster manager CR.
	if cm.Spec.RegistrationConfiguration != nil {
		o.clusterManagerChartConfig.ClusterManager.RegistrationConfiguration = *cm.Spec.RegistrationConfiguration
	}
	if cm.Spec.WorkConfiguration != nil {
		o.clusterManagerChartConfig.ClusterManager.WorkConfiguration = *cm.Spec.WorkConfiguration
	}
	if cm.Spec.AddOnManagerConfiguration != nil {
		o.clusterManagerChartConfig.ClusterManager.AddOnManagerConfiguration = *cm.Spec.AddOnManagerConfiguration
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

	crds, raw, err := chart.RenderClusterManagerChart(
		o.clusterManagerChartConfig,
		"open-cluster-management")
	if err != nil {
		return err
	}

	if err := r.ApplyRaw(crds); err != nil {
		return err
	}

	if !o.ClusteradmFlags.DryRun {
		if err := wait.WaitUntilCRDReady(
			o.Streams.Out, apiExtensionsClient, "clustermanagers.operator.open-cluster-management.io", o.wait); err != nil {
			return err
		}
	}

	if err := r.ApplyRaw(raw); err != nil {
		return err
	}

	if o.wait && !o.ClusteradmFlags.DryRun {
		if err := wait.WaitUntilRegistrationOperatorReady(
			o.Streams.Out,
			o.ClusteradmFlags.KubectlFactory,
			int64(o.ClusteradmFlags.Timeout)); err != nil {
			return err
		}
	}

	fmt.Fprint(o.Streams.Out, "upgraded completed successfully\n")
	return nil
}
