// Copyright Contributors to the Open Cluster Management project
package version

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	clusteradm "open-cluster-management.io/clusteradm"
	version "open-cluster-management.io/clusteradm/pkg/helpers/version"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	return nil
}

func (o *Options) validate() error {
	return nil
}

func (o *Options) run() (err error) {
	fmt.Printf("client\t\tversion\t:%s\n", strings.Trim(clusteradm.GetVersion(), "\n"))
	discoveryClient, err := o.ClusteradmFlags.KubectlFactory.ToDiscoveryClient()
	if err != nil {
		return err
	}
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return err
	}
	fmt.Printf("server release\tversion\t:%s\n", serverVersion.GitVersion)

	bundleVersion := version.GetDefaultBundleVersion()
	fmt.Printf("default bundle\tversion\t:%s\n", bundleVersion)
	return nil
}
