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
# Install built-in add-ons to the hub cluster
%[1]s install hub-addon --names argocd
%[1]s install hub-addon --names argocd-agent
%[1]s install hub-addon --names governance-policy-framework
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *cobra.Command {
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

	cmd.Flags().StringVar(&o.names, "names", "", "Names of the add-ons to install (comma separated) from the "+chartRepoURL+" repository. The built-in add-ons are: argocd, argocd-agent, governance-policy-framework")
	cmd.Flags().StringVar(&o.namespace, "namespace", "", "Namespace in which to install the add-ons. If not provided, the add-ons will be installed according to their chart.")
	cmd.Flags().BoolVar(&o.createNamespace, "create-namespace", false, "If true, automatically create the specified namespace.")
	cmd.Flags().StringVar(&o.chartVersion, "chart-version", "", "The chart version to use when deploying the hub add-on(s) (e.g. 0.6.0). Defaults to an empty string, which will fetch the latest released version.")

	return cmd
}
