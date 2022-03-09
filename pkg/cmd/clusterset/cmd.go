// Copyright Contributors to the Open Cluster Management project
package clusterset

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/cmd/clusterset/add"
	"open-cluster-management.io/clusteradm/pkg/cmd/clusterset/bind"
	"open-cluster-management.io/clusteradm/pkg/cmd/clusterset/remove"
	"open-cluster-management.io/clusteradm/pkg/cmd/clusterset/unbind"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// NewCmd provides a cobra command wrapping NewCmdImportCluster
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clusterset",
		Short: "clusterset sub-command",
	}

	cmd.AddCommand(add.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(bind.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(unbind.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(remove.NewCmd(clusteradmFlags, streams))

	return cmd
}
