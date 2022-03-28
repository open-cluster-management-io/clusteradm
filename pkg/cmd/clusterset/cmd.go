// Copyright Contributors to the Open Cluster Management project
package clusterset

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/cmd/clusterset/bind"
	"open-cluster-management.io/clusteradm/pkg/cmd/clusterset/set"
	"open-cluster-management.io/clusteradm/pkg/cmd/clusterset/unbind"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// NewCmd provides a cobra command wrapping NewCmdImportCluster
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clusterset",
		Short: "clusterset options",
		Long:  "there are 3 clusterset options: set, bind and unbind",
	}

	cmd.AddCommand(set.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(bind.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(unbind.NewCmd(clusteradmFlags, streams))

	return cmd
}
