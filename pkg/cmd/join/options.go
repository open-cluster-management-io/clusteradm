// Copyright Contributors to the Open Cluster Management project
package join

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

//Options: The structure holding all the command-line options
type Options struct {
	//ClusteradmFlags: The generic optiosn from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//The token generated on the hub to access it from the cluster
	token string
	//The external hub apiserver url (https://<host>:<port>)
	hubAPIServer string
	//the name under the cluster must be imported
	clusterName string

	values Values
	//The file to output the resources will be sent to the file.
	outputFile string

	//Installing release version of OCM
	version string
	//Pulling image registry of OCM
	registry string
	//Runs the cluster joining in foreground
	wait bool

	// The installing registration agent will be starting registration using
	// the external endpoint from --hub-apiserver instead of looking for the
	// internal endpoint from the public cluster-info.
	skipHubInClusterEndpointLookup bool
}

//Values: The values used in the template
type Values struct {
	//ClusterName: the name of the joined cluster on the hub
	ClusterName string
	//Hub: Hub information
	Hub Hub
	//ImageRegistry is the registry related configuration
	ImageRegistry ImageRegistry
	//ImageRegistry is the klusterlet related configuration
	Klusterlet Klusterlet
}

//Hub: The hub values for the template

type Hub struct {
	//APIServer: The API Server external URL
	APIServer string
	//KubeConfig: The kubeconfig of the boostrap secret to connect to the hub
	KubeConfig string
}

// Klusterlet is for templating klusterlet configuration
type Klusterlet struct {
	//APIServer: The API Server external URL
	APIServer string
}

type ImageRegistry struct {
	// image registry name
	Registry string
	// image version
	Version string
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
	}
}
