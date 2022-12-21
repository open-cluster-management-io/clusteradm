// Copyright Contributors to the Open Cluster Management project
package preflight

import (
	"strings"

	"github.com/pkg/errors"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

type KlusterletApiserverCheck struct {
	KlusterletApiserver string
}

func (c KlusterletApiserverCheck) Check() (warningList []string, errorList []error) {
	if !validAPIHost(c.KlusterletApiserver) {
		return nil, []error{errors.New("ConfigMap/cluster-info.data.kubeconfig.clusters[0].cluster.server field in namespace kube-public should start with http:// or https://, please edit it first")}
	}
	return nil, nil
}

func (c KlusterletApiserverCheck) Name() string {
	return "KlusterletApiserver check"
}

type HubKubeconfigCheck struct {
	Config *clientcmdapiv1.Config
}

func (c HubKubeconfigCheck) Check() (warningList []string, errorList []error) {
	if c.Config == nil {
		return nil, []error{errors.New("no hubconfig found")}
	}

	if len(c.Config.Clusters) != 1 {
		return nil, []error{errors.New("error cluster length")}
	}

	// validate apiserver foramt
	if !validAPIHost(c.Config.Clusters[0].Cluster.Server) {
		return nil, []error{errors.New("--hub-apiserver should start with http:// or https://")}
	}
	// validate ca
	if c.Config.Clusters[0].Cluster.CertificateAuthorityData == nil {
		return nil, []error{errors.New("no ca detected, creating hub kubeconfig without ca")}
	}

	// validate kubeconfig
	discoveryClient, err := helpers.CreateDiscoveryClientFromClientcmdapiv1Config(*c.Config)
	if err != nil {
		return nil, []error{err}
	}

	_, err = discoveryClient.ServerVersion()
	if err != nil {
		return nil, []error{err}

	}
	return nil, nil
}

func (c HubKubeconfigCheck) Name() string {
	return "HubKubeconfig check"
}

// utils
func validAPIHost(host string) bool {
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return true
	}
	return false
}
