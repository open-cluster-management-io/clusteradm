// Copyright Contributors to the Open Cluster Management project
package proxy

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/cmd/proxy/health"
	"open-cluster-management.io/clusteradm/pkg/cmd/proxy/kubectl"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// NewCmd provides a cobra command wrapping NewCmdImportCluster
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "helper commands for cluster-proxy addon",
	}

	cmd.AddCommand(health.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(kubectl.NewCmd(clusteradmFlags, streams))

	return cmd
}
