// Copyright Contributors to the Open Cluster Management project
package unjoin

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

var example = `
# UnJoin a cluster from a hub
%[1]s unjoin --cluster-name <cluster_name>

# UnJoin and cleanup ManagedCluster from hub
%[1]s unjoin --cluster-name <cluster_name> --cleanup-hub --hub-kubeconfig <hub_kubeconfig_path>
`

// NewCmd ...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "unjoin",
		Short:        "unjoin from a hub",
		Long:         "unjoin specific cluster from a hub cluster",
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
	cmd.Flags().StringVar(&o.clusterName, "cluster-name", "", "The name of the joining cluster")
	cmd.Flags().BoolVar(&o.purgeOperator, "purge-operator", true, "Purge the operator")
	cmd.Flags().StringVar(&o.outputFile, "output-file", "", "The generated resources will be copied in the specified file")
	cmd.Flags().BoolVar(&o.cleanupHub, "cleanup-hub", false, "Cleanup ManagedCluster resource from hub")
	cmd.Flags().StringVar(&o.hubKubeconfig, "hub-kubeconfig", "", "Path to hub kubeconfig file (required when --cleanup-hub is enabled)")
	return cmd
}
