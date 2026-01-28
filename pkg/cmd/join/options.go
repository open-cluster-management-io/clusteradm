// Copyright Contributors to the Open Cluster Management project
package join

import (
	"k8s.io/cli-runtime/pkg/genericiooptions"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"open-cluster-management.io/clusteradm/pkg/clusterprovider/capi"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

// Options: The structure holding all the command-line options
type Options struct {
	// ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	// Values below are input from flags
	// The token generated on the hub to access it from the cluster
	token string
	// The external hub apiserver url (https://<host>:<port>)
	hubAPIServer string
	// The grpc server of the hub cluster
	grpcServer string

	// The hub ca-file(optional)
	caFile string

	// The grpc ca file which can be found in the configmap ca-bundle-configmap in open-cluster-management-hub ns
	grpcCAFile string

	// the name under the cluster must be imported
	clusterName string

	// OCM agent deploy mode, default to "default".
	mode string
	// managed cluster kubeconfig file, used in hosted mode
	managedKubeconfigFile string
	// Pulling image registry of OCM
	registry string

	// imagePullCredFile is a credential file is used to pull image which should be docker credentials json file and
	// will be filled into the secret open-cluster-management-image-pull-credentials.
	imagePullCredFile string

	// version of predefined compatible image versions
	bundleVersion string
	// Path to a file containing version bundle configuration
	versionBundleFile string

	// if set, deploy the singleton agent rather than klusterlet
	singleton bool

	// The file to output the resources will be sent to the file.
	outputFile string
	// Runs the cluster joining in foreground
	wait bool
	// By default, The installing registration agent will be starting registration using
	// the external endpoint from --hub-apiserver instead of looking for the internal
	// endpoint from the public cluster-info.
	forceHubInClusterEndpointLookup bool
	// By default, the klusterlet running in the hosting cluster will access the managed
	// cluster registered in the hosted mode by using the external endpoint from
	// --managed-cluster-kubeconfig instead of looking for the internal endpoint from the
	// public cluster-info.
	forceManagedInClusterEndpointLookup bool
	hubInClusterEndpoint                string

	// Values below are temporary data
	// HubCAData: data in hub ca file
	HubCAData []byte
	// hub config
	HubConfig *clientcmdapiv1.Config

	// grpcCAData: ca data used by the GRPC server
	grpcCAData []byte

	// The URL of a forward proxy server which will be used by agents on the managed cluster
	// to connect to the hub cluster (optional)
	proxyURL string
	// The proxy server ca-file(optional)
	proxyCAFile string

	// Resource requirement for the containers managed by klusterlet and the klusterlet operator
	resourceQosClass string
	resourceLimits   map[string]string
	resourceRequests map[string]string

	// If create ns or use existing ns
	createNameSpace bool

	klusterletChartConfig *chart.KlusterletChartConfig

	capiOptions *capi.CAPIOptions

	Streams genericiooptions.IOStreams

	// enableSyncLabels is to enable the feature which can sync the labels from klusterlet to all agent resources.
	enableSyncLabels bool

	clientCertExpirationSeconds int32

	// The type of authentication to use for registering and authenticating with hub
	registrationAuth string

	// The arn of hub cluster(i.e. EKS) to which managed-cluster will join
	hubClusterArn string

	// The arn of managed cluster(i.e. EKS) which will be joining the hub
	managedClusterArn string

	// Annotations for registration controller to set on the ManagedCluster
	klusterletAnnotations []string

	// klusterletValuesFile is the path to a YAML file containing klusterlet Helm chart values.
	// The values from the file override the default klusterlet chart values.
	klusterletValuesFile string

	// The type of authentication to use for kubeClient type addon registration
	addonKubeClientRegistrationAuth string

	// Token expiration seconds for addon registration
	addonTokenExpirationSeconds int64
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags:       clusteradmFlags,
		Streams:               streams,
		capiOptions:           capi.NewCAPIOption(clusteradmFlags.KubectlFactory),
		klusterletChartConfig: chart.NewDefaultKlusterletChartConfig(),
	}
}
