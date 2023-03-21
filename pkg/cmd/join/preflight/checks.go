// Copyright Contributors to the Open Cluster Management project
package preflight

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

const (
	InstallModeDefault = "Default"
	InstallModeHosted  = "Hosted"
)

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
	if !ValidAPIHost(c.Config.Clusters[0].Cluster.Server) {
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

type DeployModeCheck struct {
	Mode                  string
	InternalEndpoint      bool
	ManagedKubeconfigFile string
}

func (c DeployModeCheck) Check() (warningList []string, errorList []error) {
	if c.Mode != InstallModeDefault && c.Mode != InstallModeHosted {
		return nil, []error{errors.New("deploy mode should be default or hosted")}
	}
	if c.Mode == InstallModeDefault {
		if c.ManagedKubeconfigFile != "" {
			return nil, []error{errors.New("--managed-cluster-kubeconfig should not be set in default deploy mode")}
		}
	} else { // c.Mode == InstallModeHosted
		if c.ManagedKubeconfigFile == "" {
			return nil, []error{errors.New("--managed-cluster-kubeconfig should be set in hosted deploy mode")}
		}
		// if we use kind cluster as managed cluster, the kubeconfig should be --internal, the kubeconfig can be used by klusterlet
		// deployed in management cluster, but can not be used by clusteradm to validate. so we jump the validate process
		if !c.InternalEndpoint {
			err := helpers.ValidateKubeconfigFile(c.ManagedKubeconfigFile)
			if err != nil {
				return nil, []error{errors.New(fmt.Sprintf("validate managed kubeconfig file failed: %v", err))}
			}
		}
	}
	return nil, nil
}

func (c DeployModeCheck) Name() string {
	return "DeployMode Check"
}

type ClusterNameCheck struct {
	ClusterName string
}

func (c ClusterNameCheck) Check() (warningList []string, errorList []error) {
	re := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	matched := re.MatchString(c.ClusterName)
	if !matched {
		return nil, []error{errors.New("validate ClusterName failed: should match `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`")}
	}
	return nil, nil
}

func (c ClusterNameCheck) Name() string {
	return "ClusterName Check"
}

// utils
func ValidAPIHost(host string) bool {
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return true
	}
	return false
}
