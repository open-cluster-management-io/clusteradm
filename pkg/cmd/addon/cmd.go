// Copyright Contributors to the Open Cluster Management project
package addon

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/cmd/addon/disable"
	"open-cluster-management.io/clusteradm/pkg/cmd/addon/enable"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// NewCmd provides a cobra command wrapping NewCmdImportCluster
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addon",
		Short: "addon options",
		Long:  "there are 2 addon options: enable and disable",
	}

	cmd.AddCommand(enable.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(disable.NewCmd(clusteradmFlags, streams))

	return cmd
}
