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
	join_scenario "open-cluster-management.io/clusteradm/pkg/cmd/join/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"open-cluster-management.io/clusteradm/pkg/helpers/wait"
	"open-cluster-management.io/clusteradm/pkg/version"
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

	versionBundle, err := version.GetVersionBundle(o.bundleVersion)
	if err != nil {
		klog.Errorf("unable to retrieve version: %v", err)
		return err
	}

	klog.V(1).InfoS("init options:", "dry-run", o.ClusteradmFlags.DryRun)
	o.values = join_scenario.Values{
		Registry:    o.registry,
		ClusterName: k.Spec.ClusterName,
		Klusterlet: join_scenario.Klusterlet{
			Name:                k.Name,
			Mode:                string(k.Spec.DeployOption.Mode),
			KlusterletNamespace: k.Spec.Namespace,
		},
		BundleVersion: join_scenario.BundleVersion{
			RegistrationImageVersion: versionBundle.Registration,
			PlacementImageVersion:    versionBundle.Placement,
			WorkImageVersion:         versionBundle.Work,
			OperatorImageVersion:     versionBundle.Operator,
		},
	}

	// reconstruct values from the klusterlet CR.
	if k.Spec.RegistrationConfiguration != nil {
		o.values.RegistrationConfiguration.RegistrationFeatures = k.Spec.RegistrationConfiguration.FeatureGates
		o.values.RegistrationConfiguration.ClientCertExpirationSeconds = k.Spec.RegistrationConfiguration.ClientCertExpirationSeconds
	}
	if k.Spec.WorkConfiguration != nil {
		o.values.WorkFeatures = k.Spec.WorkConfiguration.FeatureGates
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
	r := reader.NewResourceReader(o.ClusteradmFlags.KubectlFactory, o.ClusteradmFlags.DryRun, o.Streams)

	_, apiExtensionsClient, _, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	files := []string{
		"join/namespace.yaml",
		"join/cluster_role.yaml",
		"join/cluster_role_binding.yaml",
		"join/klusterlets.crd.yaml",
		"join/service_account.yaml",
	}

	err = r.Apply(join_scenario.Files, o.values, files...)
	if err != nil {
		return err
	}

	err = r.Apply(join_scenario.Files, o.values, "join/operator.yaml")
	if err != nil {
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

	err = r.Apply(join_scenario.Files, o.values, "join/klusterlets.cr.yaml")
	if err != nil {
		return err
	}

	fmt.Fprint(o.Streams.Out, "upgraded completed successfully\n")

	return nil
}
