// Copyright Contributors to the Open Cluster Management project
package klusterlet

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

var example = `
# Upgrade clustermanager
%[1]s upgrade clustermanager --bundle-version latest
`

// NewCmd ...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "klusterlet",
		Short:        "use this command to upgrade the klusterlet.",
		Long:         "use this command to upgrade the klusterlet.",
		Example:      fmt.Sprintf(example, helpers.GetExampleHeader()),
		SilenceUsage: true,
		PreRun: func(c *cobra.Command, args []string) {
			helpers.DryRunMessage(o.ClusteradmFlags.DryRun)
		},
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.complete(c, args); err != nil {
				return err
			}
			if err := o.validate(); err != nil {
				return err
			}
			if err := o.run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&o.registry, "image-registry", "quay.io/open-cluster-management",
		"The name of the image registry serving OCM images, which will be applied to all the deploying OCM components.")
	cmd.Flags().StringVar(&o.bundleVersion, "bundle-version", "default",
		"The version of predefined compatible image versions (e.g. v0.6.0). Defaults to the latest released version. You can also set \"latest\" to install the latest development version.")
	cmd.Flags().StringVar(&o.versionBundleFile, "bundle-version-overrides", "",
		"Path to a file containing version bundle overrides. Optional. If provided, overrides component versions within the selected version bundle.")
	cmd.Flags().BoolVar(&o.wait, "wait", false,
		"If set, the command will initialize the OCM control plan in foreground.")
	cmd.Flags().StringVar(&o.klusterletValuesFile, "klusterlet-values-file", "", "The path to a YAML file containing klusterlet Helm chart values. The values from the file override both the default klusterlet chart values and the values from other flags.")
	return cmd
}
