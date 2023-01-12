// Copyright Contributors to the Open Cluster Management project
package upgrade

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/cmd/upgrade/clustermanager"
	"open-cluster-management.io/clusteradm/pkg/cmd/upgrade/klusterlet"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// NewCmd provides a cobra command wrapping NewCmdImportCluster
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "upgrade component",
	}

	cmd.AddCommand(klusterlet.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(clustermanager.NewCmd(clusteradmFlags, streams))

	return cmd
}
