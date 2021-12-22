package common

import "k8s.io/apimachinery/pkg/util/sets"

const (
	AddonName             = "cluster-proxy"
	AddonFullName         = "open-cluster-management:cluster-proxy"
	AddonInstallNamespace = "open-cluster-management-" + AddonName

	ComponentNameProxyAgentServer = "proxy-agent-server"
	ComponentNameProxyServer      = "proxy-server"
	ComponentNameProxyAgent       = "proxy-agent"
	ComponentNameProxyClient      = "proxy-client"
)

var (
	AllComponentNames = sets.NewString(
		ComponentNameProxyAgentServer,
		ComponentNameProxyServer,
		ComponentNameProxyClient,
	)
)

const (
	SubjectGroupClusterProxy      = "open-cluster-management:cluster-proxy"
	SubjectUserClusterProxyAgent  = "open-cluster-management:cluster-proxy:proxy-agent"
	SubjectUserClusterProxyServer = "open-cluster-management:cluster-proxy:proxy-server"
	SubjectUserClusterAgentServer = "open-cluster-management:cluster-proxy:agent-server"
	SubjectUserClusterAddonAgent  = "open-cluster-management:cluster-proxy:addon-agent"
)

const (
	LabelKeyComponentName = "proxy.open-cluster-management.io/component-name"
)
