// Copyright Contributors to the Open Cluster Management project
package init

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

//Options: The structure holding all the command-line options
type Options struct {
	//ClusteradmFlags: The generic optiosn from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	values          Values
	//The file to output the resources will be sent to the file.
	outputFile string
	//If true the bootstrap token will be used instead of the service account token
	useBootstrapToken bool
	//if true the hub will be reinstalled
	force bool
	//Installing release tag of OCM
	tag string
	//Pulling image registry of OCM
	registry string
	//If set, will be persisting the generated join command to a local file
	outputJoinCommandFile string
	//If set, the command will hold until the OCM control plane initialized
	wait bool
}

//Valus: The values used in the template
type Values struct {
	//The values related to the hub
	Hub Hub `json:"hub"`
}

//Hub: The hub values for the template
type Hub struct {
	//TokenID: A token id allowing the cluster to connect back to the hub
	TokenID string `json:"tokenID"`
	//TokenSecret: A token secret allowing the cluster to connect back to the hub
	TokenSecret string `json:"tokenSecret"`
	// Image is the required configuration for workload images.
	Image Image `json:"image"`
}

type Image struct {
	// Registry is the name of the image registry to pull.
	Registry string `json:"registry"`
	// Tag is the image tag.
	Tag string `json:"tag"`
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
	}
}
