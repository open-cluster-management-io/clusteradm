// Copyright Contributors to the Open Cluster Management project
package work

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/utils/ptr"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	ClusterOption *genericclioptionsclusteradm.ClusterOption

	Streams genericiooptions.IOStreams

	Placement string

	Workname string

	FileNameFlags genericclioptions.FileNameFlags

	Overwrite bool

	UseReplicaSet bool
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
		ClusterOption:   genericclioptionsclusteradm.NewClusterOption().AllowUnset(),
		FileNameFlags: genericclioptions.FileNameFlags{
			Filenames: &[]string{},
			Recursive: ptr.To[bool](true),
		},
	}
}
