// Copyright Contributors to the Open Cluster Management project
package clustermanager

import (
	"k8s.io/cli-runtime/pkg/genericiooptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

// Options is holding all the command-line options
type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	clusterManagerChartConfig *chart.ClusterManagerChartConfig
	//The file to output the resources will be sent to the file.
	registry string
	//version of predefined compatible image versions
	bundleVersion string
	// Path to a file containing version bundle configuration
	versionBundleFile string
	//If set, the command will hold until the OCM control plane initialized
	wait bool

	// clusterManagerValuesFile is the path to a YAML file containing cluster-manager Helm chart values.
	// The values from the file override the default chart values, values reconstructed from the hub, and values from other flags.
	clusterManagerValuesFile string

	Streams genericiooptions.IOStreams
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags:           clusteradmFlags,
		Streams:                   streams,
		clusterManagerChartConfig: chart.NewDefaultClusterManagerChartConfig(),
	}
}
