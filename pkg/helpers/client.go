// Copyright Contributors to the Open Cluster Management project

package helpers

import (
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func GetControllerRuntimeClientFromFlags(configFlags *genericclioptions.ConfigFlags) (client crclient.Client, err error) {
	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return crclient.New(config, crclient.Options{})
}
