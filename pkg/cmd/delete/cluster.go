// Copyright Contributors to the Open Cluster Management project
package delete

import (
	"fmt"
	"path/filepath"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/apply"

	"github.com/open-cluster-management/cm-cli/pkg/bindata"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var deleteClusteExample = `
# Delete a cluster
%[1]s cm create cluster --values values.yaml
`

const (
	deleteClusterScenarioDirectory = "scenarios/destroy"
)

var valuesTemplatePath = filepath.Join(deleteClusterScenarioDirectory, "values-template.yaml")

type DeleteClusterOptions struct {
	applierScenariosOptions *applierscenarios.ApplierScenariosOptions
	values                  map[string]interface{}
}

func newDeleteClusterOptions(streams genericclioptions.IOStreams) *DeleteClusterOptions {
	return &DeleteClusterOptions{
		applierScenariosOptions: applierscenarios.NewApplierScenariosOptions(streams),
	}
}

// NewCmdDeleteCluster ...
func NewCmdDeleteCluster(streams genericclioptions.IOStreams) *cobra.Command {
	o := newDeleteClusterOptions(streams)

	cmd := &cobra.Command{
		Use:          "cluster",
		Short:        "Delete a cluster",
		Example:      fmt.Sprintf(deleteClusteExample, "oc/kubectl"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.SetUsageTemplate(applierscenarios.UsageTempate(cmd, valuesTemplatePath))

	o.applierScenariosOptions.AddFlags(cmd.Flags())
	o.applierScenariosOptions.ConfigFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *DeleteClusterOptions) Complete(cmd *cobra.Command, args []string) (err error) {
	o.values, err = apply.ConvertValuesFileToValuesMap(o.applierScenariosOptions.ValuesPath, "")
	if err != nil {
		return err
	}

	return nil
}

func (o *DeleteClusterOptions) Validate() (err error) {
	imc, ok := o.values["managedCluster"]
	if !ok {
		return fmt.Errorf("managedCluster is missing")
	}
	mc := imc.(map[string]interface{})

	iname, ok := mc["name"]
	if !ok {
		return fmt.Errorf("cluster name is missing")
	}
	name := iname.(string)
	if len(name) == 0 {
		return fmt.Errorf("managedCluster.name not specified")
	}

	return nil
}

func (o *DeleteClusterOptions) Run() error {

	//TODO this could be simplified by just creating new templates
	// managedcluster, clusterName with only the name of the cluster.
	reader := bindata.NewBindataReader()

	applyOptions := &apply.ApplyOptions{
		OutFile:     o.applierScenariosOptions.OutFile,
		ConfigFlags: o.applierScenariosOptions.ConfigFlags,

		Delete:    true,
		Timeout:   o.applierScenariosOptions.Timeout,
		Force:     o.applierScenariosOptions.Force,
		Silent:    o.applierScenariosOptions.Silent,
		IOStreams: o.applierScenariosOptions.IOStreams,
	}

	err := applyOptions.ApplyWithValues(reader,
		filepath.Join(deleteClusterScenarioDirectory, "hub", "common", "managed_cluster_cr.yaml"),
		o.values)
	if err != nil {
		return err
	}

	return applyOptions.ApplyWithValues(reader,
		filepath.Join(deleteClusterScenarioDirectory, "hub", "common", "cluster_deployment_cr.yaml"),
		o.values)

}
