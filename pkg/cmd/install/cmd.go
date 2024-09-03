// Copyright Contributors to the Open Cluster Management project
package install

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	hubaddon "open-cluster-management.io/clusteradm/pkg/cmd/install/hubaddon"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// NewCmd provides a cobra command wrapping addon install cmd
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "install a feature",
	}

	cmd.AddCommand(hubaddon.NewCmd(clusteradmFlags, streams))

	return cmd
}
