// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"fmt"
	"os"
	"strings"

	"open-cluster-management.io/clusteradm/pkg/helpers/reader"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"open-cluster-management.io/clusteradm/pkg/cmd/install/hubaddon/scenario"
	"open-cluster-management.io/clusteradm/pkg/version"
)

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

	names := strings.Split(o.names, ",")
	for _, n := range names {
		if _, ok := scenario.AddonDeploymentFiles[n]; !ok {
			return fmt.Errorf("invalid add-on name %s", n)
		}
	}

	versionBundle, err := version.GetVersionBundle(o.bundleVersion)
	if err != nil {
		return err
	}

	o.values.BundleVersion = versionBundle

	return nil
}

func (o *Options) run() error {
	alreadyProvidedAddons := make(map[string]bool)
	addons := make([]string, 0)
	names := strings.Split(o.names, ",")
	for _, n := range names {
		if _, ok := alreadyProvidedAddons[n]; !ok {
			alreadyProvidedAddons[n] = true
			addons = append(addons, strings.TrimSpace(n))
		}
	}
	o.values.HubAddons = addons

	klog.V(3).InfoS("values:", "addon", o.values.HubAddons)

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
			return fmt.Errorf("Error deploying %s CRDs: %w", addon, err)
		}
		err = r.Apply(scenario.Files, o.values, files.ConfigFiles...)
		if err != nil {
			return fmt.Errorf("Error deploying %s dependencies: %w", addon, err)
		}
		err = r.Apply(scenario.Files, o.values, files.DeploymentFiles...)
		if err != nil {
			return fmt.Errorf("Error deploying %s deployments: %w", addon, err)
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
