package delete

import (
	"fmt"

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
	reader := bindata.NewBindataReader()

	// tp, err := templateprocessor.NewTemplateProcessor(
	// 	reader,
	// 	&templateprocessor.Options{},
	// )
	// if err != nil {
	// 	return err
	// }

	// installConfig, err := tp.TemplateResource("scenarios/createdestroy/hub/"+o.cloud+"/install_config.yaml", o.values)
	// if err != nil {
	// 	return err
	// }

	// valueic := make(map[string]interface{})
	// err = yaml.Unmarshal(installConfig, &valueic)
	// if err != nil {
	// 	return err
	// }

	// o.values["installConfig"] = valueic

	applyOptions := &apply.ApplyOptions{
		OutFile:     o.applierScenariosOptions.OutFile,
		ConfigFlags: o.applierScenariosOptions.ConfigFlags,

		Delete:    true,
		Timeout:   o.applierScenariosOptions.Timeout,
		Force:     o.applierScenariosOptions.Force,
		Silent:    o.applierScenariosOptions.Silent,
		IOStreams: o.applierScenariosOptions.IOStreams,
	}

	err := applyOptions.ApplyWithValues(reader, "scenarios/createdestroy/hub/common/managed_cluster_cr.yaml", o.values)
	if err != nil {
		return err
	}

	return applyOptions.ApplyWithValues(reader, "scenarios/createdestroy/hub/common/cluster_deployment_cr.yaml", o.values)

}
