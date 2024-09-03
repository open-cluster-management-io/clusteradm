// Copyright Contributors to the Open Cluster Management project
package klusterlet

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"open-cluster-management.io/clusteradm/pkg/helpers/wait"
	"open-cluster-management.io/clusteradm/pkg/version"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

//nolint:deadcode,varcheck
const (
	klusterletName = "klusterlet"
)

func (o *Options) complete(_ *cobra.Command, _ []string) (err error) {
	err = o.ClusteradmFlags.ValidateManagedCluster()
	if err != nil {
		return err
	}

	f := o.ClusteradmFlags.KubectlFactory
	cfg, err := f.ToRESTConfig()
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

	bundleVersion, err := version.GetVersionBundle(o.bundleVersion)
	if err != nil {
		return err
	}

	o.klusterletChartConfig.Images = chart.ImagesConfig{
		Registry: o.registry,
		Tag:      bundleVersion.OCM,
	}

	if k.Spec.ResourceRequirement != nil {
		o.klusterletChartConfig.Klusterlet = chart.KlusterletConfig{
			ClusterName:         k.Spec.ClusterName,
			Namespace:           k.Spec.Namespace,
			Mode:                k.Spec.DeployOption.Mode,
			ResourceRequirement: *k.Spec.ResourceRequirement,
		}
	}

	klog.V(1).InfoS("init options:", "dry-run", o.ClusteradmFlags.DryRun)
	// reconstruct values from the klusterlet CR.
	if k.Spec.RegistrationConfiguration != nil {
		o.klusterletChartConfig.Klusterlet.RegistrationConfiguration = *k.Spec.RegistrationConfiguration
	}
	if k.Spec.WorkConfiguration != nil {
		o.klusterletChartConfig.Klusterlet.WorkConfiguration = *k.Spec.WorkConfiguration
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

	return nil
}

func (o *Options) run() error {
	r := reader.NewResourceReader(o.ClusteradmFlags.KubectlFactory, o.ClusteradmFlags.DryRun, o.Streams)

	_, apiExtensionsClient, _, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	raw, err := chart.RenderKlusterletChart(
		o.klusterletChartConfig,
		"open-cluster-management")
	if err != nil {
		return err
	}

	if err := r.ApplyRaw(raw); err != nil {
		return err
	}

	if !o.ClusteradmFlags.DryRun {
		if err := wait.WaitUntilCRDReady(apiExtensionsClient, "klusterlets.operator.open-cluster-management.io", o.wait); err != nil {
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

	fmt.Fprint(o.Streams.Out, "upgraded completed successfully\n")

	return nil
}
