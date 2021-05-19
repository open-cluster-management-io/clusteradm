// Copyright Contributors to the Open Cluster Management project
package cluster

import (
	"context"
	"fmt"
	"path/filepath"

	appliercmd "github.com/open-cluster-management/applier/pkg/applier/cmd"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/create/cluster/scenario"

	"github.com/ghodss/yaml"
	"github.com/open-cluster-management/applier/pkg/templateprocessor"
	"github.com/open-cluster-management/cm-cli/pkg/helpers"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"
)

const (
	AWS       = "aws"
	AZURE     = "azure"
	GCP       = "gcp"
	OPENSTACK = "openstack"
	VSPHERE   = "vsphere"
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

func (o *Options) validate() (err error) {
	if o.applierScenariosOptions.OutTemplatesDir != "" {
		return nil
	}
	imc, ok := o.values["managedCluster"]
	if !ok || imc == nil {
		return fmt.Errorf("managedCluster is missing")
	}
	mc := imc.(map[string]interface{})
	icloud, ok := mc["cloud"]
	if !ok || icloud == nil {
		return fmt.Errorf("cloud type is missing")
	}
	cloud := icloud.(string)
	if cloud != AWS && cloud != AZURE && cloud != GCP && cloud != OPENSTACK && cloud != VSPHERE {
		return fmt.Errorf("supported cloud type are (%s, %s, %s, %s, %s) and got %s", AWS, AZURE, GCP, OPENSTACK, VSPHERE, cloud)
	}
	o.cloud = cloud

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
		return scenario.GetApplierScenarioResourcesReader().ExtractAssets(scenarioDirectory,
			o.applierScenariosOptions.OutTemplatesDir)
	}
	client, err := helpers.GetControllerRuntimeClientFromFlags(o.applierScenariosOptions.ConfigFlags)
	if err != nil {
		return err
	}
	return o.runWithClient(client)
}

func (o *Options) runWithClient(client crclient.Client) error {
	pullSecret := &corev1.Secret{}
	err := client.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      "pull-secret",
			Namespace: "openshift-config",
		},
		pullSecret)
	if err != nil {
		return err
	}

	ps, err := yaml.Marshal(pullSecret)
	if err != nil {
		return err
	}

	valueps := make(map[string]interface{})
	err = yaml.Unmarshal(ps, &valueps)
	if err != nil {
		return err
	}

	o.values["pullSecret"] = valueps

	reader := scenario.GetApplierScenarioResourcesReader()
	tp, err := templateprocessor.NewTemplateProcessor(
		reader,
		&templateprocessor.Options{},
	)
	if err != nil {
		return err
	}

	installConfig, err := tp.TemplateResource(
		filepath.Join(scenarioDirectory, "hub", o.cloud, "install_config.yaml"),
		o.values)
	if err != nil {
		return err
	}

	valueic := make(map[string]interface{})
	err = yaml.Unmarshal(installConfig, &valueic)
	if err != nil {
		return err
	}

	o.values["installConfig"] = valueic

	applyOptions := &appliercmd.Options{
		OutFile:     o.applierScenariosOptions.OutFile,
		ConfigFlags: o.applierScenariosOptions.ConfigFlags,

		Delete:    false,
		Timeout:   o.applierScenariosOptions.Timeout,
		Force:     o.applierScenariosOptions.Force,
		Silent:    o.applierScenariosOptions.Silent,
		IOStreams: o.applierScenariosOptions.IOStreams,
	}

	return applyOptions.ApplyWithValues(client, reader,
		filepath.Join(scenarioDirectory, "hub", "common"),
		o.values)
}
