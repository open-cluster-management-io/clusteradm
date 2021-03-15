package attach

import (
	"fmt"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type AttachClusterOptions struct {
	applierScenariosOptions *applierscenarios.ApplierScenariosOptions
}

func newAttachClusterOptions(streams genericclioptions.IOStreams) *AttachClusterOptions {
	return &AttachClusterOptions{
		applierScenariosOptions: applierscenarios.NewApplierScenariosOptions(streams),
	}
}

// NewCmdImportCluster provides a cobra command wrapping NewCmdImportCluster
func NewCmdAttachCluster(streams genericclioptions.IOStreams) *cobra.Command {
	o := newAttachClusterOptions(streams)

	cmd := &cobra.Command{
		Use:          "cluster",
		Short:        "Import a cluster",
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

func (o *AttachClusterOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

func (o *AttachClusterOptions) Validate() error {
	return nil
}

func (o *AttachClusterOptions) Run() error {
	fmt.Println("Not yet implemented")
	return nil
}
