// Copyright Contributors to the Open Cluster Management project
package list


import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic optiosn from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//A list of comma separated addon names
	//names string
	//The sepcified namespace for addon to install
	//namespace string
	//A list of comma separated cluster names
	clusters string
	//The file to output the resources will be sent to the file.
	outputFile string
	values     Values
}

//Values: The values used in the template
type Values struct {
	//addons   []string
	clusters []string
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
	}
}