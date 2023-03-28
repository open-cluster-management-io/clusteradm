// Copyright Contributors to the Open Cluster Management project
package work

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
# Create manifestwork on a specified managed cluster.
%[1]s create work work-example -f xxx.yaml --clusters cluster1

# Update manifestwork on a specified managed cluster.
%[1]s create work work-example -f xxx.yaml --clusters cluster1 --overwrite

# Create manifestwork on placement selected managed clusters.
# For example, if placement1 in default namespace select cluster1 and cluster2, 
# then the manifestwork will be created on cluster1 and cluster2.
%[1]s create work work-example -f xxx.yaml --placement default/placement1

# Reschedule manifestwork to placement newly selected clusters.
# For example, if placement1 update decision to cluster2 and cluster3, 
# then the manifestwork will be deleted from cluster1 and created on cluster3.
%[1]s create work work-example -f xxx.yaml --placement default/placement1 --overwrite
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "work",
		Short:        "create a work using resource-to-apply yaml file",
		Long:         "create a work using a file containing common kubernetes resource manifests, or a director containing a set of manifest files.",
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

	o.ClusterOption.AddFlags(cmd.Flags())
	cmd.Flags().StringVar(&o.Placement, "placement", "", "Specify an existing placement with format <namespace>/<name>")
	cmd.Flags().BoolVar(&o.Overwrite, "overwrite", false, "Overwrite the existing work if it exists already")
	o.FileNameFlags.AddFlags(cmd.Flags())

	return cmd
}
