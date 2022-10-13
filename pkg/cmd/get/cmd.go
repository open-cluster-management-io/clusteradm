// Copyright Contributors to the Open Cluster Management project
package get

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/cmd/get/addon"
	"open-cluster-management.io/clusteradm/pkg/cmd/get/cluster"
	"open-cluster-management.io/clusteradm/pkg/cmd/get/clusterset"
	"open-cluster-management.io/clusteradm/pkg/cmd/get/hubinfo"
	"open-cluster-management.io/clusteradm/pkg/cmd/get/klusterletinfo"
	"open-cluster-management.io/clusteradm/pkg/cmd/get/placement"
	"open-cluster-management.io/clusteradm/pkg/cmd/get/token"
	"open-cluster-management.io/clusteradm/pkg/cmd/get/work"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// NewCmd provides a cobra command wrapping NewCmdImportCluster
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get information from the cluster",
	}

	cmd.AddCommand(token.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(addon.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(cluster.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(clusterset.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(hubinfo.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(klusterletinfo.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(work.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(placement.NewCmd(clusteradmFlags, streams))

	return cmd
}
