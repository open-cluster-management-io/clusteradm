// Copyright Contributors to the Open Cluster Management project
package enable

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
## Application Manager

# Enable application-manager addon on the given managed clusters in the specified namespace
%[1]s addon enable --names application-manager --namespace namespace --clusters cluster1,cluster2
# Enable application-manager addon for specified clusters
%[1]s addon enable --names application-manager --clusters cluster1,cluster2

## Policy Framework

# Enable governance-policy-framework addon on to the given managed clusters in the specified namespace
%[1]s addon enable --names governance-policy-framework --namespace namespace --clusters cluster1,cluster2
# Enable governance-policy-framework addon for specified clusters
%[1]s addon enable --names governance-policy-framework --clusters cluster1,cluster2
# Enable governance-policy-framework addon for a self-managed hub cluster
%[1]s addon enable --names governance-policy-framework --annotate addon.open-cluster-management.io/on-multicluster-hub=true  --clusters hub-cluster

# Enable config-policy-controller addon on the given managed clusters in the specified namespace
%[1]s addon enable --names config-policy-controller --namespace namespace --clusters cluster1,cluster2
# Enable config-policy-controller addon for specified clusters
%[1]s addon enable --names config-policy-controller --clusters cluster1,cluster2
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "enable",
		Short:        "enable specified addon",
		Long:         "enable specific add-on(s) agent deployment to the given managed clusters",
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
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "open-cluster-management-agent-addon", "Specified namespace to addon addon")
	cmd.Flags().StringVar(&o.OutputFile, "output-file", "", "The generated resources will be copied in the specified file")
	cmd.Flags().StringSliceVar(&o.Annotate, "annotate", []string{}, "Annotations to add to the ManagedClusterAddon (eg. key1=value1,key2=value2)")

	return cmd
}
