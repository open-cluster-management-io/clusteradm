// Copyright Contributors to the Open Cluster Management project
package version

import (
	"fmt"

	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"
	clusteradm "open-cluster-management.io/clusteradm"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	return nil
}

func (o *Options) validate() error {
	return nil
}
func (o *Options) run() (err error) {
	fmt.Printf("client\t\tversion\t:%s\n", clusteradm.GetVersion())
	client, err := helpers.GetControllerRuntimeClientFromFlags(o.ConfigFlags)
	if err != nil {
		return err
	}
	return o.runWithClient(client)
}

func (o *Options) runWithClient(client crclient.Client) (err error) {
	discoveryClient, err := o.factory.ToDiscoveryClient()
	if err != nil {
		return err
	}
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return err
	}
	fmt.Printf("server release\tversion\t:%s\n", serverVersion.GitVersion)
	return nil
}
