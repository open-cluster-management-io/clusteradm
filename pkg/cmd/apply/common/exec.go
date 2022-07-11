// Copyright Contributors to the Open Cluster Management project
package common

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"
	"open-cluster-management.io/clusteradm/pkg/helpers/asset"
)

func (o *Options) Complete(cmd *cobra.Command, args []string) (err error) {
	// Convert yaml to map[string]interface
	if len(o.ValuesPath) != 0 {
		b, err := ioutil.ReadFile(o.ValuesPath)
		if err != nil {
			return err
		}
		o.Values = make(map[string]interface{})
		if err := yaml.Unmarshal(b, &o.Values); err != nil {
			return err
		}
	}
	return nil
}

func (o *Options) Validate() error {
	reader := asset.NewYamlFileReader(o.Paths)

	assetNames, err := reader.AssetNames(nil)
	if err != nil {
		return err
	}
	if len(assetNames) == 0 {
		return fmt.Errorf("no files selected")
	}
	return nil
}

func (o *Options) Run() error {
	kubeClient, err := o.ClusteradmFlags.KubectlFactory.KubernetesClientSet()
	if err != nil {
		return err
	}
	dynamicClient, err := o.ClusteradmFlags.KubectlFactory.DynamicClient()
	if err != nil {
		return err
	}

	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	apiExtensionsClient, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	applyBuilder := apply.NewApplierBuilder().
		WithClient(kubeClient, apiExtensionsClient, dynamicClient)
	applier := applyBuilder.Build()
	reader := asset.NewYamlFileReader(o.Paths)
	files, err := reader.AssetNames(nil)
	if err != nil {
		return err
	}
	output := make([]string, 0)
	switch o.ResourcesType {
	case CoreResources:
		output, err = applier.ApplyDirectly(reader, o.Values, o.ClusteradmFlags.DryRun, "", files...)
	case Deployments:
		output, err = applier.ApplyDeployments(reader, o.Values, o.ClusteradmFlags.DryRun, "", files...)
	case CustomResources:
		output, err = applier.ApplyCustomResources(reader, o.Values, o.ClusteradmFlags.DryRun, "", files...)
	}
	if err != nil {
		return err
	}
	return apply.WriteOutput(o.OutputFile, output)
}
