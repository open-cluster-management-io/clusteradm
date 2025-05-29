// Copyright Contributors to the Open Cluster Management project
package scenario

import (
	"embed"

	"open-cluster-management.io/clusteradm/pkg/version"
)

//go:embed addon
var Files embed.FS

const (
	AppMgrAddonName          = "application-manager"
	PolicyFrameworkAddonName = "governance-policy-framework"
)

type AddonDeploymentFile struct {
	ConfigFiles     []string
	DeploymentFiles []string
	CRDFiles        []string
}

// Values: The values used in the template
type Values struct {
	HubAddons []string
	// Namespace to install
	Namespace string
	// Version to install
	BundleVersion   version.VersionBundle
	CreateNamespace bool
}

var (
	AddonDeploymentFiles = map[string]AddonDeploymentFile{
		PolicyFrameworkAddonName: {
			ConfigFiles: []string{
				"addon/policy/addon-controller_clusterrole.yaml",
				"addon/policy/addon-controller_clusterrolebinding.yaml",
				"addon/policy/addon-controller_role.yaml",
				"addon/policy/addon-controller_rolebinding.yaml",
				"addon/policy/addon-controller_serviceaccount.yaml",
				"addon/policy/propagator_clusterrole.yaml",
				"addon/policy/propagator_clusterrolebinding.yaml",
				"addon/policy/propagator_role.yaml",
				"addon/policy/propagator_rolebinding.yaml",
				"addon/policy/propagator_serviceaccount.yaml",
				"addon/policy/clustermanagementaddon_configpolicy.yaml",
				"addon/policy/clustermanagementaddon_policyframework.yaml",
			},
			CRDFiles: []string{
				"addon/policy/policy.open-cluster-management.io_placementbindings.yaml",
				"addon/policy/policy.open-cluster-management.io_policies.yaml",
				"addon/policy/policy.open-cluster-management.io_policyautomations.yaml",
				"addon/policy/policy.open-cluster-management.io_policysets.yaml",
				"addon/appmgr/crd_placementrule.yaml",
			},
			DeploymentFiles: []string{
				"addon/policy/addon-controller_deployment.yaml",
				"addon/policy/propagator_deployment.yaml",
			},
		},
		AppMgrAddonName: {
			ConfigFiles: []string{
				"addon/appmgr/clustermanagementaddon_appmgr.yaml",
				"addon/appmgr/clusterrole_agent.yaml",
				"addon/appmgr/clusterrole_binding.yaml",
				"addon/appmgr/clusterrole.yaml",
				"addon/appmgr/service_account.yaml",
				"addon/appmgr/service_metrics.yaml",
				"addon/appmgr/service_operator.yaml",
				"addon/appmgr/mutatingwebhookconfiguration.yaml",
			},
			CRDFiles: []string{
				"addon/appmgr/crd_channel.yaml",
				"addon/appmgr/crd_helmrelease.yaml",
				"addon/appmgr/crd_placementrule.yaml",
				"addon/appmgr/crd_subscription.yaml",
				"addon/appmgr/crd_subscriptionstatuses.yaml",
				"addon/appmgr/crd_report.yaml",
				"addon/appmgr/crd_clusterreport.yaml",
			},
			DeploymentFiles: []string{
				"addon/appmgr/deployment_channel.yaml",
				"addon/appmgr/deployment_subscription.yaml",
				"addon/appmgr/deployment_placementrule.yaml",
				"addon/appmgr/deployment_appsubsummary.yaml",
			},
		},
	}
)
