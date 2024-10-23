// Copyright Contributors to the Open Cluster Management project
package hubinfo

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"
)

var example = `
# Get hub-info
%[1]s get hub-info
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "hub-info",
		Short:        "get hub-info",
		Long:         "get hub-info about Registration Operator and Components",
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
			if err := o.validate(args); err != nil {
				return err
			}
			if err := o.run(); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
