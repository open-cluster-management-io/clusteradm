// Copyright Contributors to the Open Cluster Management project
package init

import (
	"k8s.io/cli-runtime/pkg/genericiooptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers/helm"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

// Options is holding all the command-line options
type Options struct {
	// ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags           *genericclioptionsclusteradm.ClusteradmFlags
	clusterManagerChartConfig *chart.ClusterManagerChartConfig
	// The file to output the resources will be sent to the file.
	outputFile string
	// If true the bootstrap token will be used instead of the service account token
	useBootstrapToken bool
	// if true the hub will be reinstalled
	force bool
	// Pulling image registry of OCM
	registry string

	// imagePullCredFile is a credential file is used to pull image which should be docker credentials json file and
	// will be filled into the secret open-cluster-management-image-pull-credentials.
	imagePullCredFile string

	// version of predefined compatible image versions
	bundleVersion string

	// If set, deploy the singleton controlplane
	singleton     bool
	SingletonName string
	Helm          *helm.Helm

	// Resource requirement for the containers managed by the cluster manager and the cluster manager operator
	resourceQosClass string
	resourceLimits   map[string]string
	resourceRequests map[string]string

	// If create ns or use existing ns
	createNamespace bool

	// If set, will be persisting the generated join command to a local file
	outputJoinCommandFile string
	// If set, the command will hold until the OCM control plane initialized
	wait bool
	//
	output string

	Streams genericiooptions.IOStreams

	// The type of authentication to use for initializing the hub cluster
	registrationDrivers []string
	// The optional ARN to pass if awsirsa is one of the registrationAuths
	// and the cluster name in EKS kubeconfig doesn't contain hubClusterArn
	hubClusterArn string

	// A list of users that can be auto approve csr and auto accept to join hub cluster
	autoApprovedCSRIdentities []string
	// A list of AWS EKS ARN patterns that are accepted and whatever matches can be auto accepted to join hub cluster
	autoApprovedARNPatterns []string
	// List of tags to be added to AWS resources created by hub while processing awsirsa registration request
	awsResourceTags []string
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags:           clusteradmFlags,
		clusterManagerChartConfig: chart.NewDefaultClusterManagerChartConfig(),
		Streams:                   streams,
		Helm:                      helm.NewHelm(),
	}
}
