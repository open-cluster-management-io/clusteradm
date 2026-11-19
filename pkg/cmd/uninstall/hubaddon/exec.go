// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"

	addonclientset "open-cluster-management.io/api/client/addon/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers/parse"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"open-cluster-management.io/clusteradm/pkg/version"

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

	addonNames := parse.UniqueNames(o.names)
	if len(addonNames) == 0 {
		return fmt.Errorf("names is missing")
	}

	for _, addonName := range addonNames {
		if addonName == clusteradmhubaddoninstall.DeprecatedArgocdAgent {
			// TODO: Remove following v1.5
			klog.Warningf("'argocd-agent' is now 'argocd-agent-addon'. Update your scripts as needed.")
		} else if _, ok := clusteradmhubaddoninstall.AddonCharts[addonName]; !ok {
			klog.Warningf("provided addon name is not recognized: %s", addonName)
		}
	}

	return nil
}

func (o *Options) run(ctx context.Context) error {
	var errs []error
	for _, a := range parse.UniqueNames(o.names) {
		// TODO: Remove following v1.5
		if a == clusteradmhubaddoninstall.DeprecatedArgocdAgent {
			a = clusteradmhubaddoninstall.ArgocdAgentAddonName
		}
		if err := o.checkExistingAddon(ctx, a); err != nil {
			errs = append(errs, err)
			continue
		}
		if a == clusteradmhubaddoninstall.PolicyFrameworkAddonName {
			err := o.uninstallPolicy()
			if err != nil {
				errs = append(errs, err)
			}
		} else {
			err := o.runWithHelmClient(a)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	return utilerrors.NewAggregate(errs)
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

	fmt.Fprintf(o.Streams.Out, "Uninstalling built-in governance-policy-framework add-on from the Hub cluster...\n")

	return nil
}

func (o *Options) runWithHelmClient(addon string) error {
	addonChart, ok := clusteradmhubaddoninstall.AddonCharts[addon]
	if !ok {
		addonChart = clusteradmhubaddoninstall.AddonChart{
			ReleaseName: addon,
		}
	}

	if addonChart.ReleaseName == "" {
		addonChart.ReleaseName = addonChart.ChartName
	}

	if addonChart.Namespace == "" {
		addonChart.Namespace = o.values.Namespace
	}

	o.Helm.WithNamespace(addonChart.Namespace)
	return o.Helm.UninstallRelease(addonChart.ReleaseName)
}

func (o *Options) checkExistingAddon(ctx context.Context, name string) error {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}

	addonClient, err := addonclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	addons, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(metav1.NamespaceAll).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return err
	}

	if len(addons.Items) > 0 {
		var enabledClusters []string
		for _, addon := range addons.Items {
			enabledClusters = append(enabledClusters, addon.Namespace)
		}
		return fmt.Errorf("there are still addons for %s enabled on some clusters. "+
			"Run `clusteradm addon disable --names %s --clusters %s` to disable addons",
			name, name, strings.Join(enabledClusters, ","))
	}
	return nil
}
