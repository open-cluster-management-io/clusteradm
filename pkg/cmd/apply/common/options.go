// Copyright Contributors to the Open Cluster Management project
package common

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type ResourceType string

const CoreResources ResourceType = "core-resources"
const Deployments ResourceType = "deployments"
const CustomResources ResourceType = "custom-resources"

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//A list of Paths
	Paths         []string
	ValuesPath    string
	Values        map[string]interface{}
	ResourcesType ResourceType
	//The file to output the resources will be sent to the file.
	OutputFile string
}

func NewOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
	}
}
