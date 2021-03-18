// Copyright Contributors to the Open Cluster Management project
package detach

import (
	"fmt"
	"path/filepath"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/apply"
	"github.com/open-cluster-management/cm-cli/pkg/helpers"
	"github.com/open-cluster-management/cm-cli/pkg/resources"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var detachClusteExample = `
# Detach a cluster
%[1]s detach cluster --values values.yaml

# Detach a cluster with overwritting the cluster name
%[1]s detach cluster --values values.yaml --name mycluster
`

const (
	detachClusterScenarioDirectory = "scenarios/detach"
)

var valuesTemplatePath = filepath.Join(detachClusterScenarioDirectory, "values-template.yaml")

type DetachClusterOptions struct {
	applierScenariosOptions *applierscenarios.ApplierScenariosOptions
	clusterName             string
	values                  map[string]interface{}
}

func newDetachClusterOptions(streams genericclioptions.IOStreams) *DetachClusterOptions {
	return &DetachClusterOptions{
		applierScenariosOptions: applierscenarios.NewApplierScenariosOptions(streams),
	}
}

// NewCmdDetachCluster ...
func NewCmdDetachCluster(streams genericclioptions.IOStreams) *cobra.Command {
	o := newDetachClusterOptions(streams)

	cmd := &cobra.Command{
		Use:          "cluster",
		Short:        "detach a cluster",
		Example:      fmt.Sprintf(detachClusteExample, helpers.GetExampleHeader()),
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
	cmd.Flags().StringVar(&o.clusterName, "name", "", "Name of the cluster to import")

	o.applierScenariosOptions.AddFlags(cmd.Flags())
	o.applierScenariosOptions.ConfigFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *DetachClusterOptions) Complete(cmd *cobra.Command, args []string) (err error) {
	o.values, err = apply.ConvertValuesFileToValuesMap(o.applierScenariosOptions.ValuesPath, "")
	if err != nil {
		return err
	}

	if len(o.values) == 0 {
		return fmt.Errorf("values are missing")
	}

	return nil
}

func (o *DetachClusterOptions) Validate() error {
	if o.clusterName == "" {
		iname, ok := o.values["managedClusterName"]
		if !ok {
			return fmt.Errorf("cluster name is missing")
		}
		o.clusterName = iname.(string)
		if len(o.clusterName) == 0 {
			return fmt.Errorf("managedClusterName not specified")
		}
	}

	o.values["managedClusterName"] = o.clusterName

	return nil
}

func (o *DetachClusterOptions) Run() error {
	reader := resources.NewResourcesReader()

	applyOptions := &apply.ApplyOptions{
		OutFile:     o.applierScenariosOptions.OutFile,
		ConfigFlags: o.applierScenariosOptions.ConfigFlags,

		Delete:    true,
		Timeout:   o.applierScenariosOptions.Timeout,
		Force:     o.applierScenariosOptions.Force,
		Silent:    o.applierScenariosOptions.Silent,
		IOStreams: o.applierScenariosOptions.IOStreams,
	}

	return applyOptions.ApplyWithValues(reader,
		filepath.Join(detachClusterScenarioDirectory, "hub", "managed_cluster_cr.yaml"),
		o.values)

}
