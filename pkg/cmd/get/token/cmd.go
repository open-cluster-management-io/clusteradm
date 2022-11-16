// Copyright Contributors to the Open Cluster Management project
package token

import (
	"fmt"

	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

var example = `
# Get the bootstrap token
%[1]s get token
`

// NewCmd ...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "token",
		Short:        "get the bootstrap token",
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

	cmd.Flags().StringVar(&o.outputFile, "output-file", "", "The generated resources will be copied in the specified file")
	cmd.Flags().BoolVar(&o.useBootstrapToken, "use-bootstrap-token", false, "If set then the bootstrap token will used instead of a service account token")
	cmd.Flags().StringVarP(&o.output, "output", "o", "text", "output should be json or text")

	return cmd
}
