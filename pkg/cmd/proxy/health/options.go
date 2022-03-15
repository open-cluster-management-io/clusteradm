// Copyright Contributors to the Open Cluster Management project
package health

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

//Options: The structure holding all the command-line options
type Options struct {
	//ClusteradmFlags: The generic optiosn from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	clusters                 []string
	inClusterProxyCertLookup bool
	proxyClientCACertPath    string
	proxyClientCertPath      string
	proxyClientKeyPath       string
	proxyServerHost          string
	proxyServerPort          int

	// completed fields
	isProxyClientCertProvided    bool
	isProxyServerAddressProvided bool
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
	}
}
