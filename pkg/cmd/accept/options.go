// Copyright Contributors to the Open Cluster Management project
package accept

import (
	"k8s.io/cli-runtime/pkg/genericiooptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//A list of comma separated cluster names
	ClusterOptions *genericclioptionsclusteradm.ClusterOption
	//Wait to wait for managedcluster and CSR
	Wait bool
	//If true the csr will approve directly and check of requester will skip.
	SkipApproveCheck bool

	Values Values

	Requesters []string

	Streams genericiooptions.IOStreams
}

// Values used in the template
type Values struct {
	Clusters []string
}

func NewOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		ClusterOptions:  genericclioptionsclusteradm.NewClusterOption(),
		Streams:         streams,
	}
}
