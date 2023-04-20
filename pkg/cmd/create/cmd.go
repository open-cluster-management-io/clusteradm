// Copyright Contributors to the Open Cluster Management project
package create

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/cmd/create/clusterset"
	"open-cluster-management.io/clusteradm/pkg/cmd/create/placement"
	"open-cluster-management.io/clusteradm/pkg/cmd/create/sampleapp"
	"open-cluster-management.io/clusteradm/pkg/cmd/create/work"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// NewCmd provides a cobra command wrapping NewCmdImportCluster
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create a resource",
	}

	cmd.AddCommand(clusterset.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(work.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(placement.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(sampleapp.NewCmd(clusteradmFlags, streams))

	return cmd
}
