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
	//version of predefined compatible image versions
	bundleVersion string
	//Pulling image registry of OCM
	registry string
	//Runs the cluster joining in foreground
	wait bool

	// By default, The installing registration agent will be starting registration using
	// the external endpoint from --hub-apiserver instead of looking for the internal
	// endpoint from the public cluster-info.
	forceHubInClusterEndpointLookup bool
}

//Values: The values used in the template
type Values struct {
	//ClusterName: the name of the joined cluster on the hub
	ClusterName string
	//Hub: Hub information
	Hub Hub
	//Registry is the image registry related configuration
	Registry string
	//Klusterlet is the klusterlet related configuration
	Klusterlet Klusterlet
	//bundle version
	BundleVersion BundleVersion
}

//Hub: The hub values for the template

type Hub struct {
	//APIServer: The API Server external URL
	APIServer string
	//KubeConfig: The kubeconfig of the boostrap secret to connect to the hub
	KubeConfig string
	//image registry
	Registry string
}

// Klusterlet is for templating klusterlet configuration
type Klusterlet struct {
	//APIServer: The API Server external URL
	APIServer string
}

type BundleVersion struct {
	// registation image version
	RegistrationImageVersion string
	// placment image version
	PlacementImageVersion string
	// work image version
	WorkImageVersion string
	// operator image version
	OperatorImageVersion string
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
	}
}
