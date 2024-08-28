// Copyright Contributors to the Open Cluster Management project
package klusterlet

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	klusterletchart "open-cluster-management.io/ocm/deploy/klusterlet/chart"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

// Options is holding all the command-line options
type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	klusterletChartConfig *klusterletchart.ChartConfig

	//The file to output the resources will be sent to the file.
	registry string
	//version of predefined compatible image versions
	bundleVersion string
	//If set, the command will hold until the OCM control plane initialized
	wait bool

	Streams genericclioptions.IOStreams
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags:       clusteradmFlags,
		Streams:               streams,
		klusterletChartConfig: chart.NewDefaultKlusterletChartConfig(),
	}
}
