// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
# Install built-in add-ons to the hub cluster
%[1]s install hub-addon --names application-manager
%[1]s install hub-addon --names governance-policy-framework
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "hub-addon",
		Short:        "install hub-addon",
		Long:         "Install specific built-in add-on(s) to the hub cluster",
		Example:      fmt.Sprintf(example, clusteradmhelpers.GetExampleHeader()),
		SilenceUsage: true,
		PreRunE: func(c *cobra.Command, args []string) error {
			clusteradmhelpers.DryRunMessage(clusteradmFlags.DryRun)

			return nil
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

	cmd.Flags().StringVar(&o.names, "names", "", "Names of the built-in add-on to install (comma separated). The built-in add-ons are: application-manager, governance-policy-framework")
	cmd.Flags().StringVar(&o.values.Namespace, "namespace", "open-cluster-management", "Namespace of the built-in add-on to install. Defaults to open-cluster-management")
	cmd.Flags().StringVar(&o.outputFile, "output-file", "", "The generated resources will be copied in the specified file")
	cmd.Flags().StringVar(&o.bundleVersion, "bundle-version", "default",
		"The image version tag to use when deploying the hub add-on(s) (e.g. v0.6.0). Defaults to the latest released version. You can also set \"latest\" to install the latest development version.")

	return cmd
}
