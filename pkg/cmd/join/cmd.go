// Copyright Contributors to the Open Cluster Management project
package join

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

var example = `
# Join a cluster to the hub 
%[1]s join --hub-token <tokenID.tokenSecret> --hub-apiserver <hub_apiserver_url> --cluster-name <cluster_name>
# join a cluster to the hub with hosted mode
%[1]s join --hub-token <tokenID.tokenSecret> --hub-apiserver <hub_apiserver_url> --cluster-name <cluster_name> --mode hosted --managed-cluster-kubeconfig <managed-cluster-kubeconfig-file>
# join a cluster to the hub while the hub provided no valid CA data in kube-public namespace
%[1]s join --hub-token <tokenID.tokenSecret> --hub-apiserver <hub_apiserver_url> --cluster-name <cluster_name> --ca-file <ca-file>
`

// NewCmd ...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "join",
		Short:        "join a cluster to the hub",
		Long:         "join specific cluster to the hub cluster",
		Example:      fmt.Sprintf(example, helpers.GetExampleHeader()),
		SilenceUsage: true,
		PreRun: func(c *cobra.Command, args []string) {
			helpers.DryRunMessage(o.ClusteradmFlags.DryRun)
		},
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.complete(c, args); err != nil {
				return err
			}
			if err := o.validate(); err != nil {
				return err
			}
			if err := o.run(); err != nil {
				return err
			}

			return nil
		},
	}

	genericclioptionsclusteradm.SpokeMutableFeatureGate.AddFlag(cmd.Flags())
	cmd.Flags().StringVar(&o.token, "hub-token", "", "The token to access the hub")
	cmd.Flags().StringVar(&o.hubAPIServer, "hub-apiserver", "", "The api server url to the hub")
	cmd.Flags().StringVar(&o.caFile, "ca-file", "", "the file path to hub ca, optional")
	cmd.Flags().StringVar(&o.clusterName, "cluster-name", "", "The name of the joining cluster")
	cmd.Flags().StringVar(&o.outputFile, "output-file", "", "The generated resources will be copied in the specified file")
	cmd.Flags().StringVar(&o.registry, "image-registry", "quay.io/open-cluster-management", "The name of the image registry serving OCM images.")
	cmd.Flags().StringVar(&o.bundleVersion, "bundle-version", "default",
		"The version of predefined compatible image versions (e.g. v0.6.0). Defaults to the latest released version. You can also set \"latest\" to install the latest development version.")
	cmd.Flags().BoolVar(&o.forceHubInClusterEndpointLookup, "force-internal-endpoint-lookup", false,
		"If true, the installed klusterlet agent will be starting the cluster registration process by "+
			"looking for the internal endpoint from the public cluster-info in the hub cluster instead of from --hub-apiserver.")
	cmd.Flags().BoolVar(&o.wait, "wait", false, "If true, running the cluster registration in foreground.")
	cmd.Flags().StringVarP(&o.mode, "mode", "m", "default", "mode to deploy klusterlet, can be default or hosted")
	cmd.Flags().StringVar(&o.managedKubeconfigFile, "managed-cluster-kubeconfig", "", "To specify the directory to external managed cluster kubeconfig in hosted mode")
	return cmd
}
