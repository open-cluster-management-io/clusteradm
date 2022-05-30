// Copyright Contributors to the Open Cluster Management project
package unjoin

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

//Options: The structure holding all the command-line options
type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//The name under the cluster must be imported
	clusterName string
	//Delete the operator by default
	purgeOperator bool
	//The file to output the resources will be sent to the file.
	outputFile string
	values     Values
}
type Values struct {
	//ClusterName: the name of the joined cluster on the hub
	ClusterName string
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
	}
}
