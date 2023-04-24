// Copyright Contributors to the Open Cluster Management project
package placement

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
# Create a placement with selector.
%[1]s create placement test --label-selectors region=apac

# Create a placement with count.
%[1]s create placement test --count 2

# Create a placement with clustersets.
%[1]s create placement test --clustersets set1

# Create a placement with clustersets prioritizers
%[1]s create placement test --prioritizers BuiltIn:Steady:3,BuiltIn:ResourceAllocatableCPU:2
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "placement",
		Short:        "create a placement",
		Long:         "create a placement",
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

	cmd.Flags().BoolVar(&o.Overwrite, "overwrite", false, "Overwrite the existing work if it exists already")
	cmd.Flags().StringVar(&o.Namespace, "namespace", "default", "Namespace to bind to a clusterset")
	cmd.Flags().StringSliceVar(&o.ClusterSelector, "label-selectors", o.ClusterSelector, "Label selectors to select clusters")
	cmd.Flags().StringSliceVar(&o.ClusterSets, "clustersets", o.ClusterSets, "Cluster Sets where clusters are selected")
	cmd.Flags().StringSliceVar(&o.Prioritizers, "prioritizers", o.Prioritizers, "Prioritizers to sort and filter clusters")
	cmd.Flags().Int32Var(&o.NumOfClusters, "count", o.NumOfClusters, "Number of clusters to select")

	return cmd
}
