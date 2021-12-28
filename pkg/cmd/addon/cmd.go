// Copyright Contributors to the Open Cluster Management project
package addon

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/cmd/addon/enable"
	"open-cluster-management.io/clusteradm/pkg/cmd/addon/list"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// NewCmd provides a cobra command wrapping NewCmdImportCluster
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addon",
		Short: "addon options",
	}

	cmd.AddCommand(enable.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(list.NewCmd(clusteradmFlags, streams))

	return cmd
}
