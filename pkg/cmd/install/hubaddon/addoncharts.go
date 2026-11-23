// Copyright Contributors to the Open Cluster Management project
package hubaddon

type AddonChart struct {
	ChartName      string   `json:"chartName"`
	ReleaseName    string   `json:"releaseName"`
	Namespace      string   `json:"namespace"`
	CRDChartName   string   `json:"crdChartName"`
	CRDReleaseName string   `json:"crdReleaseName"`
	CRDs           []string `json:"crds"`
	Version        string   `json:"version"`
}

var AddonCharts = map[string][]AddonChart{
	// ArgoCD Addons
	"argocd": {{
		ChartName: "argocd-pull-integration",
		Namespace: "argocd",
	}},
	"argocd-agent": {{
		ChartName: "argocd-agent-addon",
		Namespace: "argocd",
	}},
	// Policy Framework Addons
	"governance-policy-framework": {{
		ChartName: "governance-policy-propagator",
		Namespace: "open-cluster-management",
	}, {
		ChartName: "governance-policy-addon-controller",
		Namespace: "open-cluster-management",
	}},
}

func GetAddonNames() []string {
	addonNames := make([]string, 0, len(AddonCharts))

	for addon := range AddonCharts {
		addonNames = append(addonNames, addon)
	}

	return addonNames
}
