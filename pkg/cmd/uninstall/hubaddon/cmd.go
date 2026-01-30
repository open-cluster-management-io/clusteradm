// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

var example = `
# Uninstall built-in add-ons from the hub cluster
%[1]s uninstall hub-addon --names argocd
%[1]s uninstall hub-addon --names argocd-agent
%[1]s uninstall hub-addon --names governance-policy-framework
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "hub-addon",
		Short:        "uninstall hub-addon",
		Long:         "Uninstall specific built-in add-on(s) to the hub cluster",
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

	cmd.Flags().StringVar(&o.names, "names", "", "Names of the built-in add-on to uninstall (comma separated). The built-in add-ons are: argocd, argocd-agent, governance-policy-framework")
	cmd.Flags().StringVar(&o.values.Namespace, "namespace", "open-cluster-management", "Namespace of the built-in add-on to uninstall. Defaults to open-cluster-management")

	return cmd
}
