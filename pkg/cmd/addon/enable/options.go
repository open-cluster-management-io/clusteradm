// Copyright Contributors to the Open Cluster Management project
package enable

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//A list of comma separated addon names
	Names []string
	//The sepcified namespace for addon to install
	Namespace string
	//A list of comma separated cluster names
	Clusters []string
	//The file to output the resources will be sent to the file.
	OutputFile string
	//
	Streams genericclioptions.IOStreams
}

func NewOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
	}
}
