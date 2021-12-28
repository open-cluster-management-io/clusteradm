// Copyright Contributors to the Open Cluster Management project
package listall

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic optiosn from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//The file to output the resources will be sent to the file.
	outputFile string
	values     Values
}

//Values: The values used in the template
type Values struct {
	clusters []string
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
	}
}
