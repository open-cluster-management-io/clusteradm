// Copyright Contributors to the Open Cluster Management project
package create

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
Create an addon from manifests by using AddonTemplate
%[1]s addon create helloworld -f deployment.yaml
`

// NewCmd creates a cammand to create an addon
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "create",
		Short:        "create a specified addon",
		Long:         "create a specific add-on(s) on the hub",
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

	cmd.Flags().StringVar(&o.Version, "version", "0.0.1", "Specified version of the addon")
	cmd.Flags().BoolVar(&o.Overwrite, "overwrite", false, "Overwrite the existing addon if it exists already")
	cmd.Flags().BoolVar(&o.EnableHubRegistration, "hub-registration", false, "Enable the agent to register to the hub cluster")
	cmd.Flags().StringVar(&o.ClusterRoleBindingRef, "cluster-role-bind", "", "The rolebinding to the clusterrole in "+
		"the cluster namespace for the addon agent")
	o.FileNameFlags.AddFlags(cmd.Flags())

	return cmd
}
