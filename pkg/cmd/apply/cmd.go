// Copyright Contributors to the Open Cluster Management project
package apply

import (
	"fmt"

	"open-cluster-management.io/clusteradm/pkg/cmd/apply/common"
	"open-cluster-management.io/clusteradm/pkg/cmd/apply/core"
	"open-cluster-management.io/clusteradm/pkg/cmd/apply/customresources"
	"open-cluster-management.io/clusteradm/pkg/cmd/apply/deployments"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
# Apply templates
%[1]s apply --values values.yaml --path template_path1 tempalte_path2...
`

// NewCmd ...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := common.NewOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "apply",
		Short:        "apply templates located in paths",
		Long:         "apply templates located in paths with a values.yaml, the list of path can be a path to a file or a directory",
		Example:      fmt.Sprintf(example, helpers.GetExampleHeader()),
		SilenceUsage: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			helpers.DryRunMessage(o.ClusteradmFlags.DryRun)
		},
	}

	cmd.AddCommand(core.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(customresources.NewCmd(clusteradmFlags, streams))
	cmd.AddCommand(deployments.NewCmd(clusteradmFlags, streams))
	return cmd
}
