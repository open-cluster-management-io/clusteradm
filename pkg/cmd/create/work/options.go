// Copyright Contributors to the Open Cluster Management project
package work

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	ClusterOption *genericclioptionsclusteradm.ClusterOption

	Streams genericclioptions.IOStreams

	Placement string

	Workname string

	FileNameFlags genericclioptions.FileNameFlags

	Overwrite bool
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
		ClusterOption:   genericclioptionsclusteradm.NewClusterOption().AllowUnset(),
		FileNameFlags: genericclioptions.FileNameFlags{
			Filenames: &[]string{},
			Recursive: boolPtr(true),
		},
	}
}

func boolPtr(val bool) *bool {
	return &val
}
