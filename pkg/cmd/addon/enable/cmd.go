// Copyright Contributors to the Open Cluster Management project
package enable

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
# Enable addon on to the given managed clusters of the specify namespace
%[1]s addon enable --name application-manager --namespace namespace --cluster cluster1,cluster2
# Enable application-manager addon specified clusters
%[1]s addon enable --name application-manager --cluster cluster1,cluster2
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "enable",
		Short:        "enable specified addon",
		Long:         "enable specific add-on(s) agent deployment to the given managed clusters",
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

	cmd.Flags().StringSliceVar(&o.Names, "name", []string{}, "Names of the add-on to deploy (comma separated)")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "open-cluster-management-agent-addon", "Specified namespace to addon addon")
	cmd.Flags().StringSliceVar(&o.Clusters, "cluster", []string{}, "Names of the managed cluster to deploy the add-on to (comma separated)")
	cmd.Flags().StringVar(&o.OutputFile, "output-file", "", "The generated resources will be copied in the specified file")

	return cmd
}
