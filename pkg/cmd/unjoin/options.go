// Copyright Contributors to the Open Cluster Management project
package unjoin

import (
	"k8s.io/cli-runtime/pkg/genericiooptions"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// Options is holding all the command-line options
type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//The name under the cluster must be imported
	clusterName string
	//Delete the operator by default
	purgeOperator bool
	//The file to output the resources will be sent to the file.
	outputFile string
	//Enable hub-side ManagedCluster cleanup
	cleanupHub bool
	//Path to hub kubeconfig file for hub cleanup
	hubKubeconfig string
	values        Values

	Streams genericiooptions.IOStreams
}
type Values struct {
	//ClusterName: the name of the joined cluster on the hub
	ClusterName string
	// DeployMode, KlusterletName and AgentNamespace would be auto filled
	DeployMode     operatorv1.InstallMode
	KlusterletName string
	AgentNamespace string
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
	}
}
