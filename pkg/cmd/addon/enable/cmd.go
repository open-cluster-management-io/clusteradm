// Copyright Contributors to the Open Cluster Management project
package enable

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

var example = `
## Application Manager

# Enable argocd addon (basic pull model) on the given managed clusters in the specified namespace
%[1]s addon enable --names argocd --namespace namespace --clusters cluster1,cluster2
# Enable argocd addon (basic pull model) for specified clusters
%[1]s addon enable --names argocd --clusters cluster1,cluster2

# Enable argocd-agent-addon (advanced pull model) for specified clusters
%[1]s addon enable --names argocd-agent-addon --clusters cluster1,cluster2

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

## With Configuration File

# Enable addon with configurations from a file
%[1]s addon enable --names my-addon --clusters cluster1 --config-file addon-config.yaml
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *cobra.Command {
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
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "open-cluster-management-agent-addon", "Specified namespace to deploy addon")
	cmd.Flags().StringVar(&o.OutputFile, "output-file", "", "The generated resources will be copied in the specified file")
	cmd.Flags().StringSliceVar(&o.Annotate, "annotate", []string{}, "Annotations to add to the ManagedClusterAddon (eg. key1=value1,key2=value2)")
	cmd.Flags().StringSliceVar(&o.Labels, "labels", []string{}, "Labels to add to the ManagedClusterAddon (eg. key1=value1,key2=value2)")
	cmd.Flags().StringVar(&o.ConfigFile, "config-file", "", "Path to the configuration file containing addon configs (YAML format with group, resource, namespace, and name)")

	return cmd
}
