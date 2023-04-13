// Copyright Contributors to the Open Cluster Management project
package placement

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	Streams genericclioptions.IOStreams

	Namespace string

	Placement string

	ClusterSets []string

	ClusterSelector []string

	// Prioritizers is a string array to define the prioritizers in the placement. The format is
	// Builtin:{Type}:{Weight} or Addon:{Type}:{ScoreName}:{Weight}
	Prioritizers []string

	NumOfClusters int32

	Overwrite bool
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
		Namespace:       metav1.NamespaceDefault,
	}
}
