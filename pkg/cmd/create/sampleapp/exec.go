// Copyright Contributors to the Open Cluster Management project

package sampleapp

import (
	"fmt"

	"open-cluster-management.io/clusteradm/pkg/helpers/reader"

	"github.com/spf13/cobra"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/cmd/create/sampleapp/scenario"
)

const (
	defaultSampleAppName = "sampleapp"
	pathToAppManifests   = "scenario/sampleapp"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {

	if len(args) > 1 {
		return fmt.Errorf("only one sample app name can be specified")
	}

	if len(args) == 0 {
		o.SampleAppName = defaultSampleAppName
	} else {
		o.SampleAppName = args[0]
	}

	return nil
}

func (o *Options) Validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	return nil
}

func (o *Options) Run() (err error) {

	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}

	clusterClient, err := clusterclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	return o.runWithClient(clusterClient, o.ClusteradmFlags.DryRun)
}

func (o *Options) runWithClient(clusterClient clusterclientset.Interface, dryRun bool) error {

	// Apply sample application manifest to hub cluster
	err := o.deployApp()
	if err != nil {
		return err
	}

	return nil
}

func (o *Options) deployApp() error {
	// Prepare deployment tools
	r := reader.NewResourceReader(o.ClusteradmFlags.KubectlFactory, o.ClusteradmFlags.DryRun, o.Streams)

	return r.Apply(scenario.Files, o, scenario.SampleAppFiles...)
}
