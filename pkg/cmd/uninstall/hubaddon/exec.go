// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/sets"
	"strings"

	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"open-cluster-management.io/clusteradm/pkg/version"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	clusteradmhubaddoninstall "open-cluster-management.io/clusteradm/pkg/cmd/install/hubaddon"
	"open-cluster-management.io/clusteradm/pkg/cmd/install/hubaddon/scenario"
)

func (o *Options) complete(_ *cobra.Command, _ []string) (err error) {
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

	return nil
}

func (o *Options) run() error {
	addons := sets.New[string](strings.Split(o.names, ",")...)
	if len(addons) == 0 {
		return nil
	}
	for a := range addons {
		if a == clusteradmhubaddoninstall.PolicyFrameworkAddonName {
			return o.uninstallPolicy()
		}
		return o.runWithHelmClient(a)
	}

	return o.uninstallPolicy()
}

func (o *Options) uninstallPolicy() error {

	r := reader.NewResourceReader(o.ClusteradmFlags.KubectlFactory, o.ClusteradmFlags.DryRun, o.Streams)

	files, ok := scenario.AddonDeploymentFiles[clusteradmhubaddoninstall.PolicyFrameworkAddonName]
	if !ok {
		return fmt.Errorf("no add-ons found in files")
	}

	// this needs to be set to render the manifests, but the version value
	// does not matter.
	o.values.BundleVersion, _ = version.GetVersionBundle("default", "")

	err := r.Delete(scenario.Files, o.values, files.ConfigFiles...)
	if err != nil {
		return err
	}

	err = r.Delete(scenario.Files, o.values, files.DeploymentFiles...)
	if err != nil {
		return err
	}

	fmt.Fprintf(o.Streams.Out, "Uninstalling built-in policy add-on from the Hub cluster...\n")

	return nil
}

func (o *Options) runWithHelmClient(addon string) error {
	addonChart, ok := clusteradmhubaddoninstall.AddonCharts[addon]
	if !ok {
		addonChart = clusteradmhubaddoninstall.AddonChart{
			ChartName:   addon,
			ReleaseName: addon,
		}
	}

	if addonChart.Namespace == "" {
		addonChart.Namespace = o.values.Namespace
	}

	o.Helm.WithNamespace(addonChart.Namespace)
	return o.Helm.UninstallRelease(addonChart.ReleaseName)
}
