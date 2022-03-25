// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//A list of comma separated addon names
	names string
	//The file to output the resources will be sent to the file.
	outputFile string
	values     Values
}

//Values: The values used in the template
type Values struct {
	hubAddons []string
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
	}
}
