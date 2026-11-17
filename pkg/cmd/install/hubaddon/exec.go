// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"open-cluster-management.io/clusteradm/pkg/helpers"
)

const (
	ChartRepoURL  = "https://open-cluster-management.io/helm-charts"
	chartRepoName = "ocm"
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

func (o *Options) run(ctx context.Context) error {
	addonsToInstall := sets.New[string]()
	names := strings.SplitSeq(o.names, ",")
	for n := range names {
		addonsToInstall.Insert(n)
	}

	var errs []error
	for addon := range addonsToInstall {
		if err := o.runWithHelmClient(ctx, addon); err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

func GetAddonCharts(addon string, namespace string, chartVersion string) []AddonChart {
	addonChartList, ok := AddonCharts[addon]
	if !ok {
		addonChartList = []AddonChart{{
			ChartName:   addon,
			ReleaseName: addon,
		}}
	}

	for i, addonChart := range addonChartList {
		if addonChart.ReleaseName == "" {
			addonChartList[i].ReleaseName = addonChartList[i].ChartName
		}

		if addonChart.CRDReleaseName == "" {
			addonChartList[i].CRDReleaseName = addonChartList[i].CRDChartName
		}

		if addonChart.Namespace == "" {
			addonChartList[i].Namespace = namespace
		}

		if addonChart.Version == "" {
			addonChart.Version = chartVersion
		}
	}

	return addonChartList
}

func (o *Options) runWithHelmClient(ctx context.Context, addon string) error {
	var errs []error

	for _, addonChart := range GetAddonCharts(addon, o.namespace, o.chartVersion) {
		o.Helm.WithNamespace(addonChart.Namespace)
		o.Helm.WithCreateNamespace(o.createNamespace)

		if err := o.Helm.PrepareChart(ctx, chartRepoName, ChartRepoURL); err != nil {
			errs = append(errs, err)

			continue
		}

		if o.ClusteradmFlags.DryRun {
			o.Helm.SetValue("dryRun", "true")
		}

		if addonChart.CRDReleaseName != "" {
			if err := o.installCRDChart(ctx, addonChart); err != nil {
				return err
			}
		}

		if err := o.Helm.InstallChart(ctx, addonChart.ReleaseName, chartRepoName, addonChart.ChartName, addonChart.Version); err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

func (o *Options) installCRDChart(ctx context.Context, addonChart AddonChart) error {
	// Install the CRDs from their own chart first. Helm does not reliably wait
	// for CRDs in a chart's crds/ folder before creating resources that use them.
	// Chart repos without the CRDs chart fall back to the addon chart's own crds/.
	// Uninstall keeps the CRDs release, so skip the install when it is already deployed.
	status, err := o.Helm.ReleaseStatus(addonChart.CRDReleaseName)
	if err != nil {
		return err
	}
	switch status {
	case "":
		err = o.Helm.InstallChart(ctx, addonChart.CRDReleaseName, chartRepoName, addonChart.CRDChartName, addonChart.Version)
	case release.StatusDeployed:
		// Already installed, reuse it.
	default:
		// Helm refuses to install over any existing release history.
		return fmt.Errorf("release %s in namespace %s is %s, remove it with \"helm uninstall %s -n %s\" and retry",
			addonChart.CRDReleaseName, addonChart.Namespace, status, addonChart.CRDReleaseName, addonChart.Namespace)
	}
	switch {
	case errors.Is(err, repo.ErrNoChartName):
		klog.Warningf("chart %s not found in the %s repository, installing %s with its own CRDs",
			addonChart.CRDChartName, chartRepoName, addonChart.ChartName)
	case err != nil:
		return err
	case !o.ClusteradmFlags.DryRun:
		if err := o.waitForCRDs(ctx, addonChart.CRDs); err != nil {
			return err
		}
	}
	return nil
}

// waitForCRDs waits up to about a minute per CRD. Establishing the large Argo CD
// CRDs can take longer than WaitUntilCRDReady's ~6s budget on a busy API server.
func (o *Options) waitForCRDs(ctx context.Context, crds []string) error {
	_, apiExtensionsClient, _, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}
	fmt.Fprintf(o.Streams.Out, "Waiting for CRDs to be established: %s\n", strings.Join(crds, ", "))
	b := wait.Backoff{Duration: time.Second, Factor: 1.0, Steps: 60}
	for _, crd := range crds {
		if err := helpers.WaitCRDToBeReady(ctx, apiExtensionsClient, crd, b, false); err != nil {
			return err
		}
	}
	return nil
}
