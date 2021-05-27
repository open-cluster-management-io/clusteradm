// Copyright Contributors to the Open Cluster Management project
package version

import (
	"fmt"

	"github.com/open-cluster-management/cm-cli/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var example = `
# Version
%[1]s version
`

// NewCmd provides a cobra command wrapping NewCmdImportCluster
func NewCmd(f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(f, streams)
	cmd := &cobra.Command{
		Use:          "version",
		Short:        "get the versions of the different components",
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

	return cmd
}
