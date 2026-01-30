// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/parse"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"open-cluster-management.io/clusteradm/pkg/cmd/install/hubaddon/scenario"
	"open-cluster-management.io/clusteradm/pkg/version"
)

const (
	url                      = "https://open-cluster-management.io/helm-charts"
	repoName                 = "ocm"
	ArgocdAddonName          = "argocd"
	DeprecatedArgocdAgent    = "argocd-agent"
	ArgocdAgentAddonName     = "argocd-agent-addon"
	PolicyFrameworkAddonName = "governance-policy-framework"
)

type AddonChart struct {
	ChartName      string
	ReleaseName    string
	Namespace      string
	CRDChartName   string
	CRDReleaseName string
	CRDs           []string
}

var AddonCharts = map[string]AddonChart{
	ArgocdAddonName: {
		ChartName: "argocd-pull-integration",
		Namespace: "argocd",
	},
	ArgocdAgentAddonName: {
		ChartName:    "argocd-agent-addon",
		CRDChartName: "argocd-agent-addon-crds",
		CRDs: []string{
			"argocds.argoproj.io",
			"gitopsclusters.apps.open-cluster-management.io",
		},
		Namespace: "argocd",
	},
	PolicyFrameworkAddonName: {},
}

func (o *Options) complete(_ *cobra.Command, _ []string) (err error) {
	klog.V(1).InfoS("addon options:", "dry-run", o.ClusteradmFlags.DryRun, "names", o.names, "output-file", o.outputFile)
	return nil
}

func (o *Options) validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	addonNames := parse.ParseUniqueNames(o.names)
	if len(addonNames) == 0 {
		return fmt.Errorf("names is missing")
	}

	for _, addonName := range addonNames {
		if addonName == DeprecatedArgocdAgent {
			// TODO: Remove following v1.5
			klog.Warningf("'argocd-agent' is now 'argocd-agent-addon'. Update your scripts as needed.")
		} else if _, ok := AddonCharts[addonName]; !ok {
			klog.Warningf("provided addon name is not recognized: %s", addonName)
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
	var errs []error
	for _, addon := range parse.ParseUniqueNames(o.names) {
		// TODO: Remove following v1.5
		if addon == DeprecatedArgocdAgent {
			addon = ArgocdAgentAddonName
		}
		if addon == PolicyFrameworkAddonName {
			if err := o.installPolicyAddon(ctx); err != nil {
				errs = append(errs, err)
			}
		} else {
			if err := o.runWithHelmClient(ctx, addon); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return utilerrors.NewAggregate(errs)
}

func (o *Options) installPolicyAddon(ctx context.Context) error {
	if o.values.CreateNamespace && !o.ClusteradmFlags.DryRun {
		if err := o.createNamespace(ctx); err != nil {
			return err
		}
	}

	r := reader.NewResourceReader(o.ClusteradmFlags.KubectlFactory, o.ClusteradmFlags.DryRun, o.Streams)
	files, ok := scenario.AddonDeploymentFiles[PolicyFrameworkAddonName]
	if !ok {
		return fmt.Errorf("no policy framework addon")
	}
	err := r.Apply(scenario.Files, o.values, files.CRDFiles...)
	if err != nil {
		return fmt.Errorf("error deploying %s CRDs: %w", PolicyFrameworkAddonName, err)
	}
	err = r.Apply(scenario.Files, o.values, files.ConfigFiles...)
	if err != nil {
		return fmt.Errorf("error deploying %s dependencies: %w", PolicyFrameworkAddonName, err)
	}
	err = r.Apply(scenario.Files, o.values, files.DeploymentFiles...)
	if err != nil {
		return fmt.Errorf("error deploying %s deployments: %w", PolicyFrameworkAddonName, err)
	}

	fmt.Fprintf(o.Streams.Out, "Installing built-in %s add-on to the Hub cluster...\n", PolicyFrameworkAddonName)

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
	if err != nil && k8serrors.IsNotFound(err) {
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
	addonChartToInstall, ok := AddonCharts[addon]
	if !ok {
		addonChartToInstall = AddonChart{
			ChartName:   addon,
			ReleaseName: addon,
		}
	}

	if addonChartToInstall.ReleaseName == "" {
		addonChartToInstall.ReleaseName = addonChartToInstall.ChartName
	}

	if addonChartToInstall.CRDReleaseName == "" {
		addonChartToInstall.CRDReleaseName = addonChartToInstall.CRDChartName
	}

	if addonChartToInstall.Namespace == "" {
		addonChartToInstall.Namespace = o.values.Namespace
	}

	o.Helm.WithNamespace(addonChartToInstall.Namespace)
	o.Helm.WithCreateNamespace(o.values.CreateNamespace)

	if err := o.Helm.PrepareChart(ctx, repoName, url); err != nil {
		return err
	}

	if addonChartToInstall.CRDReleaseName != "" {
		if err := o.installCRDChart(ctx, addonChartToInstall); err != nil {
			return err
		}
	}

	return o.Helm.InstallChart(ctx, addonChartToInstall.ReleaseName, repoName, addonChartToInstall.ChartName)
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
		err = o.Helm.InstallChart(ctx, addonChart.CRDReleaseName, repoName, addonChart.CRDChartName)
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
			addonChart.CRDChartName, repoName, addonChart.ChartName)
	case err != nil:
		return err
	case !o.ClusteradmFlags.DryRun && len(addonChart.CRDs) > 0:
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
