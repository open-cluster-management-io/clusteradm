// Copyright Contributors to the Open Cluster Management project
package create

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/pointer"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//Name is the addon name
	Name string

	// version is the version of the addon
	Version string

	Overwrite bool

	EnableHubRegistration bool

	// registration only supports clusterRoleBinding with cluster namespace
	ClusterRoleBindingRef string

	FileNameFlags genericclioptions.FileNameFlags
	//
	Streams genericclioptions.IOStreams
}

func NewOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
		FileNameFlags: genericclioptions.FileNameFlags{
			Filenames: &[]string{},
			Recursive: pointer.Bool(true),
		},
	}
}
