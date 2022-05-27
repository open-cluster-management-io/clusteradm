// Copyright Contributors to the Open Cluster Management project
package klusterletinfo

import (
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	Streams genericclioptions.IOStreams

	printer        printer.PrefixWriter
	operatorClient operatorclient.Interface
	kubeClient     kubernetes.Interface
	crdClient      clientset.Interface
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
		printer:         printer.NewPrefixWriter(streams.Out),
	}
}
