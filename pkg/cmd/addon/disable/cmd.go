// Copyright Contributors to the Open Cluster Management project
package disable

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
# Disable application-manager addon on specified clusters
%[1]s addon disable --names application-manager --clusters cluster1,cluster2
# Disable application-manager addon on all clusters
%[1]s addon disable --names application-manager --all-clusters
# Disable application-manager addon to the given managed clusters in the specified namespace
%[1]s addon disable --names application-manager --namespace <namespace> --clusters <cluster1>

## Policy Framework

# Disable governance-policy-framework addon on specified clusters
%[1]s addon disable --names governance-policy-framework --clusters cluster1,cluster2
# Disable governance-policy-framework addon on all clusters
%[1]s addon disable --names governance-policy-framework --all-clusters
# Disable governance-policy-framework addon to the given managed clusters in the specified namespace
%[1]s addon disable --names governance-policy-framework --namespace <namespace> --clusters <cluster1>

# Disable config-policy-controller addon on specified clusters
%[1]s addon disable --names config-policy-controller --clusters cluster1,cluster2
# Disable config-policy-controller addon on all clusters
%[1]s addon disable --names config-policy-controller --all-clusters
# Disable config-policy-controller addon to the given managed clusters in the specified namespace
%[1]s addon disable --names config-policy-controller --namespace <namespace> --clusters <cluster1>
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
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
	cmd.Flags().StringSliceVar(&o.Names, "names", []string{}, "Names of the add-on to deploy (comma separated)")

	return cmd
}
