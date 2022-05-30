// Copyright Contributors to the Open Cluster Management project
package kubectl

import (
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

// Options: only support use in-cluster certificates
type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	cluster               string
	managedServiceAccount string
	kubectlArgs           string
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
	}
}

func (o *Options) validate() error {
	if o.cluster == "" {
		return errors.Errorf("cluster is required")
	}
	if o.managedServiceAccount == "" {
		return errors.Errorf("managedServiceAccount is required")
	}
	return nil
}
