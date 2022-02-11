// Copyright Contributors to the Open Cluster Management project
package get

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/cmd/get/addon"
	"open-cluster-management.io/clusteradm/pkg/cmd/get/token"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// NewCmd provides a cobra command wrapping NewCmdImportCluster
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get a resource",
	}

	cmd.AddCommand(token.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(addon.NewCmd(clusteradmFlags, streams))

	return cmd
}
