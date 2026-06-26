// Copyright Contributors to the Open Cluster Management project
package hubaddon

import (
	"k8s.io/cli-runtime/pkg/genericiooptions"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers/helm"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//A list of comma separated addon names
	names string
	//Namespace of the built-in add-on to install
	namespace string
	//If true, automatically create the specified namespace
	createNamespace bool
	//The file to output the resources will be sent to the file.
	outputFile    string
	bundleVersion string
	// Path to a file containing version bundle configuration
	versionBundleFile string

	Streams genericiooptions.IOStreams

	Helm *helm.Helm
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
		Helm:            helm.NewHelm(clusteradmFlags, streams),
	}
}
