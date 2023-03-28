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

	ClusterOption         *genericclioptionsclusteradm.ClusterOption
	managedServiceAccount string
	kubectlArgs           string
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		ClusterOption:   genericclioptionsclusteradm.NewClusterOption(),
	}
}

func (o *Options) validate() error {
	if err := o.ClusterOption.Validate(); err != nil {
		return err
	}
	if o.managedServiceAccount == "" {
		return errors.Errorf("managedServiceAccount is required")
	}
	return nil
}
