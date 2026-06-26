// Copyright Contributors to the Open Cluster Management project
package hubaddon

type AddonChart struct {
	ChartName      string
	ReleaseName    string
	Namespace      string
	CRDChartName   string
	CRDReleaseName string
	CRDs           []string
	Version        string
}

const (
	ArgocdAddonName          = "argocd"
	ArgocdAgentAddonName     = "argocd-agent"
	PolicyFrameworkAddonName = "governance-policy-framework"
)

var AddonCharts = map[string][]AddonChart{
	// ArgoCD Addons
	ArgocdAddonName: {{
		ChartName: "argocd-pull-integration",
		Namespace: "argocd",
	}},
	ArgocdAgentAddonName: {{
		ChartName: "argocd-agent-addon",
		Namespace: "argocd",
	}},
	// Policy Framework Addons
	PolicyFrameworkAddonName: {{
		ChartName: "governance-policy-propagator",
		Namespace: "open-cluster-management",
	}, {
		ChartName: "governance-policy-addon-controller",
		Namespace: "open-cluster-management",
	}},
}
