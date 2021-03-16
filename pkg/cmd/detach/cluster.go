// Copyright Contributors to the Open Cluster Management project
package detach

import (
	"fmt"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type DetachClusterOptions struct {
	applierScenariosOptions *applierscenarios.ApplierScenariosOptions
}

func newDetachClusterOptions(streams genericclioptions.IOStreams) *DetachClusterOptions {
	return &DetachClusterOptions{
		applierScenariosOptions: applierscenarios.NewApplierScenariosOptions(streams),
	}
}

// NewCmdDetachCluster ...
func NewCmdDetachCluster(streams genericclioptions.IOStreams) *cobra.Command {
	o := newDetachClusterOptions(streams)

	cmd := &cobra.Command{
		Use:          "cluster",
		Short:        "detach a cluster",
		Example:      fmt.Sprintf(applierscenarios.ApplierScenariosExample, "kubectl"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
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

	o.applierScenariosOptions.AddFlags(cmd.Flags())
	o.applierScenariosOptions.ConfigFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *DetachClusterOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

func (o *DetachClusterOptions) Validate() error {
	return nil
}

func (o *DetachClusterOptions) Run() error {
	fmt.Println("Not yet implemented")
	return nil
}
