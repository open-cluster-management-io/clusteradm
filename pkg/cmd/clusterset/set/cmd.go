// Copyright Contributors to the Open Cluster Management project
package set

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

var example = `
# Set clusters to a clusterset
%[1]s clusterset set clusterset1 --clusters cluster1,cluster2
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *cobra.Command {
	o := NewOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "set clusters or managed namespaces to a clusterset",
		Long: "after setting clusters or managed namespaces to a clusterset, the clusterset will contain the specified resources " +
			"and in order to operate that clusterset we are supposed to bind it to an existing namespace",
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
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&o.Clusters, "clusters", []string{}, "Names of the managed cluster to set to the clusterset (comma separated)")
	cmd.Flags().StringSliceVar(&o.Namespaces, "namespaces", []string{}, "Managed namespaces to bind the clusterset to (comma separated)")

	return cmd
}
