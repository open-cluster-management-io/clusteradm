// Copyright Contributors to the Open Cluster Management project
package disable

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
# Disable application-manager addon on speccified clusters
%[1]s addon disable --name application-manager --cluster cluster1,cluster2
# Disable application-manager addon on all clusters
%[1]s addon disable --name application-manager --all-clusters
# Disable application-manager addon to the given managed clusters of the specify namespace
%[1]s addon disable --names application-manager --namespace <namespace> --cluster <cluster1>
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "disable",
		Short:        "disable specified addon on specified managed clusters",
		Long:         "disable specific add-on(s) agent deployment to the given managed clusters",
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

	cmd.Flags().StringSliceVar(&o.names, "name", []string{}, "Names of the add-on to deploy (comma separated)")
	cmd.Flags().StringSliceVar(&o.clusters, "cluster", []string{}, "Names of the managed cluster to deploy the add-on to (comma separated)")
	cmd.Flags().BoolVar(&o.allclusters, "all-clusters", false, "Make all managed clusters to disable the add-on")

	return cmd
}
