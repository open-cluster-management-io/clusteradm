// Copyright Contributors to the Open Cluster Management project
package disable

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

var example = `
# Disable argocd addon (basic pull model) on specified cluster
%[1]s addon disable --names argocd --cluster cluster1
# Disable argocd addon (basic pull model) on multiple clusters
%[1]s addon disable --names argocd --clusters cluster1,cluster2

# Disable argocd-agent-addon (advanced pull model) on specified cluster
%[1]s addon disable --names argocd-agent-addon --cluster cluster1
# Disable argocd-agent-addon (advanced pull model) on multiple clusters
%[1]s addon disable --names argocd-agent-addon --clusters cluster1,cluster2

## Policy Framework

# Disable governance-policy-framework addon on specified cluster
%[1]s addon disable --names governance-policy-framework --cluster cluster1
# Disable governance-policy-framework addon on multiple clusters
%[1]s addon disable --names governance-policy-framework --clusters cluster1,cluster2

# Disable config-policy-controller addon on specified cluster
%[1]s addon disable --names config-policy-controller --cluster cluster1
# Disable config-policy-controller addon on multiple clusters
%[1]s addon disable --names config-policy-controller --clusters cluster1,cluster2
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *cobra.Command {
	o := NewOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "disable",
		Short:        "disable specified addon on specified managed clusters",
		Long:         "disable specific add-on(s) agent deployment to the given managed clusters",
		Example:      fmt.Sprintf(example, clusteradmhelpers.GetExampleHeader()),
		SilenceUsage: true,
		PreRunE: func(c *cobra.Command, args []string) error {
			clusteradmhelpers.DryRunMessage(clusteradmFlags.DryRun)

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	o.ClusterOptions.AddFlags(cmd.Flags())
	cmd.Flags().StringSliceVar(&o.Names, "names", []string{}, "Names of the add-on to disable (comma separated)")

	return cmd
}
