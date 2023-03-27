// Copyright Contributors to the Open Cluster Management project
package join

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// Options: The structure holding all the command-line options
type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	//Values below are input from flags
	//The token generated on the hub to access it from the cluster
	token string
	//The external hub apiserver url (https://<host>:<port>)
	hubAPIServer string
	//The hub ca-file(optional)
	caFile string
	//the name under the cluster must be imported
	clusterName string
	// klusterlet deploy mode, supported values are "hosted" and "default"
	mode string
	// managed cluster kubeconfig file, used in hosted mode
	managedKubeconfigFile string

	//Pulling image registry of OCM
	registry string
	// version of predefined compatible image versions
	bundleVersion string
	//The file to output the resources will be sent to the file.
	outputFile string
	//Runs the cluster joining in foreground
	wait bool
	// By default, The installing registration agent will be starting registration using
	// the external endpoint from --hub-apiserver instead of looking for the internal
	// endpoint from the public cluster-info.
	forceHubInClusterEndpointLookup bool
	hubInClusterEndpoint            string

	//Values below are tempoary data
	//HubCADate: data in hub ca file
	HubCADate []byte
	// hub config
	HubConfig *clientcmdapiv1.Config

	//Values below are used to fill in yaml files
	values Values
}

// Values: The values used in the template
type Values struct {
	//ClusterName: the name of the joined cluster on the hub
	ClusterName string
	//Hub: Hub information
	Hub Hub
	//Klusterlet is the klusterlet related configuration
	Klusterlet Klusterlet
	//Registry is the image registry related configuration
	Registry string
	//bundle version
	BundleVersion BundleVersion
	// managed kubeconfig
	ManagedKubeconfig string

	// Features is the slice of feature for registration
	RegistrationFeatures []operatorv1.FeatureGate

	// Features is the slice of feature for work
	WorkFeatures []operatorv1.FeatureGate
}

// Hub: The hub values for the template
type Hub struct {
	//APIServer: The API Server external URL
	APIServer string
	//KubeConfig: The kubeconfig of the bootstrap secret to connect to the hub
	KubeConfig string
}

// Klusterlet is for templating klusterlet configuration
type Klusterlet struct {
	//APIServer: The API Server external URL
	APIServer           string
	Mode                string
	Name                string
	AgentNamespace      string
	KlusterletNamespace string
}

type BundleVersion struct {
	// registration image version
	RegistrationImageVersion string
	// placement image version
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
