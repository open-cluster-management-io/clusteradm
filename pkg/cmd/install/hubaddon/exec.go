// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"context"
	"fmt"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	argocdAgentAddonName     = "argocd-agent"
	policyFrameworkAddonName = "governance-policy-framework"
)

type addonChart struct {
	chartName   string
	releaseName string
	namespace   string
	version     string
}

var addonCharts = map[string]addonChart{
	argocdAddonName: {
		chartName:   "argocd-pull-integration",
		releaseName: "argocd-pull-integration",
		namespace:   "argocd",
	},
	argocdAgentAddonName: {
		chartName:   "argocd-agent-addon",
		releaseName: "argocd-agent-addon",
		namespace:   "argocd",
	},
}

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
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

	versionBundle, err := version.GetVersionBundle(o.bundleVersion, o.versionBundleFile)
	if err != nil {
		return err
	}

	o.values.BundleVersion = versionBundle

	return nil
}

func (o *Options) run() error {
	addonsToInstall := sets.New[string]()
	names := strings.Split(o.names, ",")
	for _, n := range names {
		addonsToInstall.Insert(n)
	}

	var errs []error
	for addon := range addonsToInstall {
		if addon == policyFrameworkAddonName {
			if err := o.installPolicyAddon(); err != nil {
				errs = append(errs, err)
			}
		} else {
			if err := o.runWithHelmClient(addon); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return utilerrors.NewAggregate(errs)
}

func (o *Options) installPolicyAddon() error {
	if o.values.CreateNamespace {
		if err := o.createNamespace(); err != nil {
			return err
		}
	}

	r := reader.NewResourceReader(o.ClusteradmFlags.KubectlFactory, o.ClusteradmFlags.DryRun, o.Streams)
	files, ok := scenario.AddonDeploymentFiles[policyFrameworkAddonName]
	if !ok {
		return fmt.Errorf("no policy framework addon")
	}
	err := r.Apply(scenario.Files, o.values, files.CRDFiles...)
	if err != nil {
		return fmt.Errorf("Error deploying %s CRDs: %w", policyFrameworkAddonName, err)
	}
	err = r.Apply(scenario.Files, o.values, files.ConfigFiles...)
	if err != nil {
		return fmt.Errorf("Error deploying %s dependencies: %w", policyFrameworkAddonName, err)
	}
	err = r.Apply(scenario.Files, o.values, files.DeploymentFiles...)
	if err != nil {
		return fmt.Errorf("Error deploying %s deployments: %w", policyFrameworkAddonName, err)
	}

	fmt.Fprintf(o.Streams.Out, "Installing built-in %s add-on to the Hub cluster...\n", policyFrameworkAddonName)

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

func (o *Options) createNamespace() error {
	clientSet, err := o.ClusteradmFlags.KubectlFactory.KubernetesClientSet()
	if err != nil {
		return fmt.Errorf("failed to create kubernetes clientSet")
	}

	ns, err := clientSet.CoreV1().Namespaces().Get(context.Background(), o.values.Namespace, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		ns, err = clientSet.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
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

func (o *Options) runWithHelmClient(addon string) error {
	addonChartToInstall, ok := addonCharts[addon]
	if !ok {
		addonChartToInstall = addonChart{
			chartName:   addon,
			releaseName: addon,
			namespace:   o.values.Namespace,
		}
	}

	if addonChartToInstall.namespace != "" {
		o.Helm.WithNamespace(addonChartToInstall.namespace)
	}
	o.Helm.WithCreateNamespace(o.values.CreateNamespace)

	if err := o.Helm.PrepareChart(repoName, url); err != nil {
		return err
	}

	if o.ClusteradmFlags.DryRun {
		o.Helm.SetValue("dryRun", "true")
	}

	o.Helm.InstallChart(addonChartToInstall.releaseName, repoName, addonChartToInstall.chartName)

	return nil
}
