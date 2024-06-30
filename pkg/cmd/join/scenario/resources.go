// Copyright Contributors to the Open Cluster Management project
package scenario

import (
	"embed"

	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/pkg/helpers/resourcerequirement"
)

//go:embed join bootstrap_hub_kubeconfig.yaml
var Files embed.FS

// Values: The values used in the template
type Values struct {
	// ClusterName: the name of the joined cluster on the hub
	ClusterName string
	// AgentNamespace: the namespace to deploy the agent
	AgentNamespace string
	// Hub: Hub information
	Hub Hub
	// Klusterlet is the klusterlet related configuration
	Klusterlet Klusterlet
	// Registry is the image registry related configuration
	Registry string

	// ImagePullCred is the credential used to pull image. should be a base64 string and will be filled into the
	// default image pull secret named open-cluster-management-image-pull-credentials.
	ImagePullCred string

	// bundle version
	BundleVersion BundleVersion
	// managed kubeconfig
	ManagedKubeconfig string

	RegistrationConfiguration struct {
		// Features is the slice of feature for registration
		RegistrationFeatures []operatorv1.FeatureGate

		// clientCertExpirationSeconds is the expiration time for the client certificate
		ClientCertExpirationSeconds int32
	}

	// Features is the slice of feature for work
	WorkFeatures []operatorv1.FeatureGate

	// ResourceRequirement is the resource requirement setting for the containers managed by the klusterlet
	// and the klusterlet operator
	ResourceRequirement resourcerequirement.ResourceRequirement

	// EnableSyncLabels is to enable the feature which can sync the labels from klusterlet to all agent resources.
	EnableSyncLabels bool
}

// Hub: The hub values for the template
type Hub struct {
	// APIServer: The API Server external URL
	APIServer string
	// KubeConfig: The kubeconfig of the bootstrap secret to connect to the hub
	KubeConfig string
}

// Klusterlet is for templating klusterlet configuration
type Klusterlet struct {
	// APIServer: The API Server external URL
	APIServer           string
	Mode                string
	Name                string
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
