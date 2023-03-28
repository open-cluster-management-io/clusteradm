// Copyright Contributors to the Open Cluster Management project
package work

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// Options is holding all the command-line options
type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	ClusterOptions *genericclioptionsclusteradm.ClusterOption

	Streams genericclioptions.IOStreams

	Workname string

	Force bool
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
		ClusterOptions:  genericclioptionsclusteradm.NewClusterOption(),
	}
}
