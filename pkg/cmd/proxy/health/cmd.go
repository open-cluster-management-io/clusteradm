// Copyright Contributors to the Open Cluster Management project
package health

import (
	"fmt"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
# Probing healthiness of each managed clusters through the konnectivity tunnels installed by cluster-proxy addon
%[1]s proxy health
`

const (
	addrLocalhost string = "127.0.0.1"
)

// NewCmd ...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "health",
		Short:        "show the overall healthiness of cluster-proxy addon",
		Long:         "check the healthiness of a certain managed cluster that have cluster-proxy addon",
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
			if err := o.run(streams); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&o.inClusterProxyCertLookup, "in-cluster-proxy-cert-lookup", true,
		"If true, will be looking for the proxy client credentials (including CA cert, client client and key) "+
			"from the ManagedProxyConfiguration in the hub cluster")
	cmd.Flags().StringVar(&o.proxyClientCACertPath, "proxy-ca-cert", "",
		"The path to proxy server's CA certificate")
	cmd.Flags().StringVar(&o.proxyClientCertPath, "proxy-cert", "",
		"The path to proxy server's corresponding client certificate")
	cmd.Flags().StringVar(&o.proxyClientCertPath, "proxy-key", "",
		"The path to proxy server's corresponding client key")
	cmd.Flags().StringVar(&o.proxyServerHost, "proxy-server-host", addrLocalhost,
		"Konnectivity proxy server's entry hostname")
	cmd.Flags().IntVar(&o.proxyServerPort, "proxy-server-port", 8090,
		"Konnectivity proxy server's entry port")
	o.ClusterOption.AddFlags(cmd.Flags())

	return cmd
}
