// Copyright Contributors to the Open Cluster Management project
package clustermanager

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

var example = `
# Upgrade clustermanager
%[1]s upgrade clustermanager --bundle-version latest
`

// NewCmd ...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "clustermanager",
		Short:        "use this command to upgrade the clustermanager.",
		Long:         "use this command to upgrade the clustermanager.",
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
	cmd.Flags().BoolVar(&o.wait, "wait", false,
		"If set, the command will initialize the OCM control plan in foreground.")
	return cmd
}
