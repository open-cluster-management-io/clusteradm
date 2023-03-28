// Copyright Contributors to the Open Cluster Management project
package disable

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	ClusterOptions  *genericclioptionsclusteradm.ClusterOption
	//A list of comma separated addon names
	Names []string
	//The specified namespace for addon to disable
	Namespace string

	Streams genericclioptions.IOStreams
}

func NewOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
		ClusterOptions:  genericclioptionsclusteradm.NewClusterOption().AllowUnset(),
	}
}
