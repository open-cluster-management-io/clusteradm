// Copyright Contributors to the Open Cluster Management project
package work

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic optiosn from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	Streams genericclioptions.IOStreams

	Cluster string

	Workname string

	FileNameFlags genericclioptions.FileNameFlags

	Overwrite bool
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
		Cluster:         "",
		FileNameFlags: genericclioptions.FileNameFlags{
			Filenames: &[]string{},
			Recursive: boolPtr(true),
		},
	}
}

func boolPtr(val bool) *bool {
	return &val
}
