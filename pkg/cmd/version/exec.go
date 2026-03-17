// Copyright Contributors to the Open Cluster Management project
package version

import (
	"fmt"

	"github.com/spf13/cobra"
	"open-cluster-management.io/clusteradm/pkg/version"
)

func (o *Options) complete(_ *cobra.Command, _ []string) (err error) {
	return nil
}

func (o *Options) validate() error {
	return nil
}

func (o *Options) run() (err error) {
	bundleVersion := version.GetDefaultBundleVersion()

	fmt.Printf("clusteradm\tversion\t:%s\n", version.Get().GitVersion)
	fmt.Printf("default bundle\tversion\t:%s\n", bundleVersion)

	return nil
}
