// Copyright Contributors to the Open Cluster Management project
package init

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers/helm"
)

// Options is holding all the command-line options
type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	values          Values
	//The file to output the resources will be sent to the file.
	outputFile string
	//If true the bootstrap token will be used instead of the service account token
	useBootstrapToken bool
	//if true the hub will be reinstalled
	force bool
	//Pulling image registry of OCM
	registry string
	//version of predefined compatible image versions
	bundleVersion string

	//If set, deploy the singleton controlplane
	singleton     bool
	SingletonName string
	Helm          *helm.Helm

	//If set, will be persisting the generated join command to a local file
	outputJoinCommandFile string
	//If set, the command will hold until the OCM control plane initialized
	wait bool
	//
	output string

	builder *resource.Builder

	Streams genericclioptions.IOStreams
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
	// addon manager image version
	AddonManagerImageVersion string
}

// Values: The values used in the template
type Values struct {
	//The values related to the hub
	Hub Hub `json:"hub"`
	//bundle version
	BundleVersion BundleVersion

	// if enable auto approve
	AutoApprove bool

	// Features is the slice of feature for registration
	RegistrationFeatures []operatorv1.FeatureGate

	// Features is the slice of feature for work
	WorkFeatures []operatorv1.FeatureGate

	// Features is the slice of feature for addon manager
	AddonFeatures []operatorv1.FeatureGate
}

// Hub: The hub values for the template
type Hub struct {
	//TokenID: A token id allowing the cluster to connect back to the hub
	TokenID string `json:"tokenID"`
	//TokenSecret: A token secret allowing the cluster to connect back to the hub
	TokenSecret string `json:"tokenSecret"`
	// Registry is the name of the image registry to pull.
	Registry string `json:"registry"`
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
		Helm:            helm.NewHelm(),
	}
}
