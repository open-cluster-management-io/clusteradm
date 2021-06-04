// Copyright Contributors to the Open Cluster Management project
package accept

import (
	"fmt"

	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var example = `
# Accept clusters
%[1]s accept --clusters <cluster_1>,<cluster_2>,...
`

// NewCmd ...
func NewCmd(f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(f, streams)

	cmd := &cobra.Command{
		Use:          "accept",
		Short:        "accept a list of clusters",
		Example:      fmt.Sprintf(example, helpers.GetExampleHeader()),
		SilenceUsage: true,
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

	cmd.Flags().StringVar(&o.clusters, "clusters", "", "Names of the cluster to accept (comma separated)")

	return cmd
}
