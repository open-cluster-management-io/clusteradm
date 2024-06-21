// Copyright Contributors to the Open Cluster Management project
package uninstall

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/cmd/uninstall/hubaddon"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// NewCmd provides a cobra command wrapping addon uninstall cmd
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "uninstall a feature",
	}

	cmd.AddCommand(hubaddon.NewCmd(clusteradmFlags, streams))

	return cmd
}
