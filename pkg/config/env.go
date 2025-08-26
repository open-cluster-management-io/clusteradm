// Copyright Contributors to the Open Cluster Management project

package config

const (
	OpenClusterManagementNamespace    = "open-cluster-management"
	BootstrapSAName                   = "agent-registration-bootstrap"
	BootstrapClusterRoleBindingName   = "open-cluster-management:bootstrap:agent-registration"
	BootstrapClusterRoleBindingSAName = "agent-registration-bootstrap"
	BootstrapClusterRoleName          = "open-cluster-management:bootstrap"
	ClusterManagerName                = "cluster-manager"
	KlusterletName                    = "klusterlet"
	LabelApp                          = "app"
	BootstrapSecretPrefix             = "bootstrap-token-"
	HubClusterNamespace               = "open-cluster-management-hub"
	ManagedClusterNamespace           = "open-cluster-management-agent"
	ManagedProxyConfigurationName     = "cluster-proxy"
	ImagePullSecret                   = "open-cluster-management-image-pull-credentials"
)
