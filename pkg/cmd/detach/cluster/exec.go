// Copyright Contributors to the Open Cluster Management project
package cluster

import (
	"fmt"
	"path/filepath"

	appliercmd "github.com/open-cluster-management/applier/pkg/applier/cmd"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/detach/cluster/scenario"
	"github.com/open-cluster-management/cm-cli/pkg/helpers"

	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	if o.applierScenariosOptions.OutTemplatesDir != "" {
		return nil
	}
	o.values, err = appliercmd.ConvertValuesFileToValuesMap(o.applierScenariosOptions.ValuesPath, "")
	if err != nil {
		return err
	}

	if len(o.values) == 0 {
		return fmt.Errorf("values are missing")
	}

	return nil
}

func (o *Options) validate() error {
	if o.applierScenariosOptions.OutTemplatesDir != "" {
		return nil
	}
	imc, ok := o.values["managedCluster"]
	if !ok || imc == nil {
		return fmt.Errorf("managedCluster is missing")
	}
	mc := imc.(map[string]interface{})

	if o.clusterName == "" {
		iname, ok := mc["name"]
		if !ok || iname == nil {
			return fmt.Errorf("cluster name is missing")
		}
		o.clusterName = iname.(string)
		if len(o.clusterName) == 0 {
			return fmt.Errorf("managedCluster.name not specified")
		}
	}

	mc["name"] = o.clusterName

	return nil
}

func (o *Options) run() error {
	if o.applierScenariosOptions.OutTemplatesDir != "" {
		reader := scenario.GetApplierScenarioResourcesReader()
		return reader.ExtractAssets(scenarioDirectory, o.applierScenariosOptions.OutTemplatesDir)
	}
	client, err := helpers.GetControllerRuntimeClientFromFlags(o.applierScenariosOptions.ConfigFlags)
	if err != nil {
		return err
	}
	return o.runWithClient(client)
}

func (o *Options) runWithClient(client crclient.Client) error {
	reader := scenario.GetApplierScenarioResourcesReader()

	applyOptions := &appliercmd.Options{
		OutFile:     o.applierScenariosOptions.OutFile,
		ConfigFlags: o.applierScenariosOptions.ConfigFlags,

		Delete:    true,
		Timeout:   o.applierScenariosOptions.Timeout,
		Force:     o.applierScenariosOptions.Force,
		Silent:    o.applierScenariosOptions.Silent,
		IOStreams: o.applierScenariosOptions.IOStreams,
	}

	return applyOptions.ApplyWithValues(client, reader,
		filepath.Join(scenarioDirectory, "hub"),
		o.values)

}
