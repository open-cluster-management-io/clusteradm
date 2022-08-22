// Copyright Contributors to the Open Cluster Management project

package init

import (
	"fmt"

	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

var example = `
# Clean up the resource from the init stage
%[1]s clean
`

// NewCmd ...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "clean",
		Short:        "clean the hub",
		Long:         "clean up the hub control plane - cluster manager resource is deleted first",
		Example:      fmt.Sprintf(example, helpers.GetExampleHeader()),
		SilenceUsage: true,
		PreRun: func(c *cobra.Command, args []string) {
			helpers.DryRunMessage(o.ClusteradmFlags.DryRun)
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

	cmd.Flags().StringVar(&o.ClusterManageName, "name", "cluster-manager", "The name of the cluster manager resource")
	cmd.Flags().StringVar(&o.OutputFile, "output-file", "", "The generated resources will be copied in the specified file")
	cmd.Flags().BoolVar(&o.purgeOperator, "purge-operator", true, "Purge the operator")
	return cmd
}
