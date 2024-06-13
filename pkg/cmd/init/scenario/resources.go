// Copyright Contributors to the Open Cluster Management project
package scenario

import (
	"embed"

	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/pkg/helpers/resourcerequirement"
)

//go:embed init
var Files embed.FS

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
	// The values related to the hub
	Hub Hub `json:"hub"`
	// bundle version
	BundleVersion BundleVersion

	// if enable auto approve
	AutoApprove bool

	// Features is the slice of feature for registration
	RegistrationFeatures []operatorv1.FeatureGate

	// Features is the slice of feature for work
	WorkFeatures []operatorv1.FeatureGate

	// Features is the slice of feature for addon manager
	AddonFeatures []operatorv1.FeatureGate

	// ResourceRequirement is the resource requirement setting for the containers managed by the cluster manager
	// and the cluster manager operator
	ResourceRequirement resourcerequirement.ResourceRequirement
}

// Hub: The hub values for the template
type Hub struct {
	// TokenID: A token id allowing the cluster to connect back to the hub
	TokenID string `json:"tokenID"`
	// TokenSecret: A token secret allowing the cluster to connect back to the hub
	TokenSecret string `json:"tokenSecret"`
	// Registry is the name of the image registry to pull.
	Registry string `json:"registry"`

	// ImagePullCred is the credential used to pull image. should be a base64 string and will be filled into the default image pull secret
	// named open-cluster-management-image-pull-credentials.
	ImagePullCred string `json:"imagePullCred"`
}
