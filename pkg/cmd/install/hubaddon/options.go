// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"open-cluster-management.io/clusteradm/pkg/cmd/install/hubaddon/scenario"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//A list of comma separated addon names
	names string
	//The file to output the resources will be sent to the file.
	outputFile    string
	values        scenario.Values
	bundleVersion string

	Streams genericiooptions.IOStreams
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
	}
}
