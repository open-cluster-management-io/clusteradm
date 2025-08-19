// Copyright Contributors to the Open Cluster Management project
package enable

import (
	"k8s.io/cli-runtime/pkg/genericiooptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	// ClusterOptions is the option for setting clusters
	ClusterOptions *genericclioptionsclusteradm.ClusterOption
	//A list of comma separated addon names
	Names []string
	//The specified namespace for addon to install
	Namespace string
	//The file to output the resources will be sent to the file.
	OutputFile string
	//Annotations to add to the addon
	Annotate []string
	//Labels to add to the addon
	Labels []string
	//
	Streams genericiooptions.IOStreams
}

func NewOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
		ClusterOptions:  genericclioptionsclusteradm.NewClusterOption(),
	}
}
