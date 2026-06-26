// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"k8s.io/cli-runtime/pkg/genericiooptions"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers/helm"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//A list of comma separated addon names
	names string
	// The namespace in which to install the hub add-on(s)
	namespace string
	// Whether to create the hub add-on namespace during install
	createNamespace bool
	// The chart version to use when deploying the hub add-on(s)
	chartVersion string

	Streams genericiooptions.IOStreams

	Helm *helm.Helm
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
		Helm:            helm.NewHelm(),
	}
}
