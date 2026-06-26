// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"fmt"
	"strings"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

const (
	chartRepoURL  = "https://open-cluster-management.io/helm-charts"
	chartRepoName = "ocm"
)

func (o *Options) complete(_ *cobra.Command, _ []string) (err error) {
	klog.V(1).InfoS("addon options:", "dry-run", o.ClusteradmFlags.DryRun, "names", o.names, "output-file", o.outputFile)
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
	addonsToInstall := sets.New[string]()
	names := strings.SplitSeq(o.names, ",")
	for n := range names {
		addonsToInstall.Insert(n)
	}

	var errs []error
	for addon := range addonsToInstall {
		if err := o.runWithHelmClient(addon); err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

func GetAddonCharts(addon string, namespace string, chartVersion string) []AddonChart {
	addonChart, ok := AddonCharts[addon]
	if !ok {
		addonChart = []AddonChart{{
			ChartName:   addon,
			ReleaseName: addon,
		}}
	}

	for _, addonChart := range addonChart {
		if addonChart.Namespace == "" {
			addonChart.Namespace = namespace
		}

		if addonChart.Version == "" {
			addonChart.Version = chartVersion
		}
	}

	return addonChart
}

func (o *Options) runWithHelmClient(addon string) error {
	var errs []error

	for _, addonChart := range GetAddonCharts(addon, o.namespace, o.chartVersion) {
		o.Helm.WithNamespace(addonChart.Namespace)
		o.Helm.WithCreateNamespace(o.createNamespace)

		if err := o.Helm.PrepareChart(chartRepoName, chartRepoURL); err != nil {
			errs = append(errs, err)

			continue
		}

		if o.ClusteradmFlags.DryRun {
			o.Helm.SetValue("dryRun", "true")
		}

		o.Helm.InstallChart(addonChart.ReleaseName, chartRepoName, addonChart.ChartName, addonChart.Version)
	}

	return utilerrors.NewAggregate(errs)
}
