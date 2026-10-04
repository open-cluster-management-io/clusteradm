// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"context"
	stderrors "errors"
	"fmt"
	"os"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"open-cluster-management.io/clusteradm/pkg/cmd/install/hubaddon/scenario"
	"open-cluster-management.io/clusteradm/pkg/version"
)

var (
	url                      = "https://open-cluster-management.io/helm-charts"
	repoName                 = "ocm"
	argocdAddonName          = "argocd"
	argocdNamespace          = "argocd"
	argocdReleaseName        = "argocd-pull-integration"
	argocdChartName          = "argocd-pull-integration"
	argocdAgentAddonName     = "argocd-agent"
	argocdAgentReleaseName   = "argocd-agent-addon"
	argocdAgentChartName     = "argocd-agent-addon"
	argocdAgentCRDsRelease   = "argocd-agent-addon-crds"
	argocdAgentCRDsChart     = "argocd-agent-addon-crds"
	policyFrameworkAddonName = "governance-policy-framework"

	// CRDs used by resources in the argocd-agent-addon chart. They must be
	// established before that chart is installed.
	argocdAgentCRDs = []string{
		"argocds.argoproj.io",
		"gitopsclusters.apps.open-cluster-management.io",
	}
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

	names := strings.Split(o.names, ",")
	for _, n := range names {
		if n != argocdAddonName && n != argocdAgentAddonName && n != policyFrameworkAddonName {
			return fmt.Errorf("invalid add-on name %s", n)
		}
	}

	versionBundle, err := version.GetVersionBundle(o.bundleVersion, o.versionBundleFile)
	if err != nil {
		return err
	}

	o.values.BundleVersion = versionBundle

	return nil
}

func (o *Options) run(ctx context.Context) error {
	alreadyProvidedAddons := make(map[string]bool)
	addons := make([]string, 0)
	names := strings.Split(o.names, ",")
	for _, n := range names {
		if _, ok := alreadyProvidedAddons[n]; !ok {
			alreadyProvidedAddons[n] = true
			addons = append(addons, strings.TrimSpace(n))
		}
	}

	var filteredAddons []string
	for _, a := range addons {
		if a == argocdAddonName || a == argocdAgentAddonName {
			if err := o.runWithHelmClient(ctx, a); err != nil {
				return err
			}
		} else {
			filteredAddons = append(filteredAddons, a)
		}
	}
	addons = filteredAddons
	if len(addons) == 0 {
		return nil
	}

	o.values.HubAddons = addons

	klog.V(3).InfoS("values:", "addon", o.values.HubAddons)

	if o.values.CreateNamespace {
		if err := o.createNamespace(ctx); err != nil {
			return err
		}
	}

	return o.runWithClient()
}

func (o *Options) runWithClient() error {

	r := reader.NewResourceReader(o.ClusteradmFlags.KubectlFactory, o.ClusteradmFlags.DryRun, o.Streams)

	for _, addon := range o.values.HubAddons {
		files, ok := scenario.AddonDeploymentFiles[addon]
		if !ok {
			continue
		}
		err := r.Apply(scenario.Files, o.values, files.CRDFiles...)
		if err != nil {
			return fmt.Errorf("error deploying %s CRDs: %w", addon, err)
		}
		err = r.Apply(scenario.Files, o.values, files.ConfigFiles...)
		if err != nil {
			return fmt.Errorf("error deploying %s dependencies: %w", addon, err)
		}
		err = r.Apply(scenario.Files, o.values, files.DeploymentFiles...)
		if err != nil {
			return fmt.Errorf("error deploying %s deployments: %w", addon, err)
		}

		fmt.Fprintf(o.Streams.Out, "Installing built-in %s add-on to the Hub cluster...\n", addon)
	}

	if len(o.outputFile) > 0 {
		sh, err := os.OpenFile(o.outputFile, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(sh, "%s", string(r.RawAppliedResources()))
		if err != nil {
			return err
		}
		if err := sh.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (o *Options) createNamespace(ctx context.Context) error {
	clientSet, err := o.ClusteradmFlags.KubectlFactory.KubernetesClientSet()
	if err != nil {
		return fmt.Errorf("failed to create kubernetes clientSet")
	}

	ns, err := clientSet.CoreV1().Namespaces().Get(ctx, o.values.Namespace, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		ns, err = clientSet.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: o.values.Namespace,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create namespace %s: %w", ns, err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", ns, err)
	}

	return nil
}

func (o *Options) runWithHelmClient(ctx context.Context, addon string) error {
	o.Helm.WithCreateNamespace(o.values.CreateNamespace)

	if addon == argocdAddonName {
		o.Helm.WithNamespace(argocdNamespace)

		if err := o.Helm.PrepareChart(ctx, repoName, url); err != nil {
			return err
		}

		err := o.Helm.InstallChart(ctx, argocdReleaseName, repoName, argocdChartName)
		if err != nil {
			return err
		}
	}

	if addon == argocdAgentAddonName {
		o.Helm.WithNamespace(argocdNamespace)

		if err := o.Helm.PrepareChart(ctx, repoName, url); err != nil {
			return err
		}

		// Install the CRDs from their own chart first. Helm does not reliably wait
		// for CRDs in a chart's crds/ folder before creating resources that use them.
		// Chart repos without the CRDs chart fall back to the addon chart's own crds/.
		err := o.Helm.InstallChart(ctx, argocdAgentCRDsRelease, repoName, argocdAgentCRDsChart)
		switch {
		case stderrors.Is(err, repo.ErrNoChartName):
			klog.Warningf("chart %s not found in the %s repository, installing %s with its own CRDs",
				argocdAgentCRDsChart, repoName, argocdAgentChartName)
		case err != nil:
			return err
		case !o.ClusteradmFlags.DryRun:
			if err := o.waitForCRDs(ctx, argocdAgentCRDs); err != nil {
				return err
			}
		}

		err = o.Helm.InstallChart(ctx, argocdAgentReleaseName, repoName, argocdAgentChartName)
		if err != nil {
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
