// Copyright Contributors to the Open Cluster Management project

package sampleapp

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic optiosn from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	//
	Streams genericclioptions.IOStreams

	// The base name for the resources created for this sample app
	SampleAppName string

	// The specified namespace for sameple app deployment
	Namespace string

	//The file to output the resources will be sent to the file.
	OutputFile string

	builder *resource.Builder
}

func NewOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
	}
}
