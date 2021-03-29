// Copyright Contributors to the Open Cluster Management project
package create

import (
	"fmt"
	"path/filepath"

	"github.com/open-cluster-management/cm-cli/pkg/helpers"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/create/cluster/scenario"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	scenarioDirectory = "create"
)

var valuesTemplatePath = filepath.Join(scenarioDirectory, "values-template.yaml")

var example = `
# Create a cluster
%[1]s create cluster --values values.yaml

# Create a cluster
%[1]s create cluster --values values.yaml
`

// NewCmd ...
func NewCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(streams)

	cmd := &cobra.Command{
		Use:          "cluster",
		Short:        "Create a cluster",
		Example:      fmt.Sprintf(example, helpers.GetExampleHeader()),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.complete(c, args); err != nil {
				return err
			}
			if err := o.validate(); err != nil {
				return err
			}
			if err := o.run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.SetUsageTemplate(applierscenarios.UsageTempate(cmd, scenario.GetApplierScenarioResourcesReader(), valuesTemplatePath))
	cmd.Flags().StringVar(&o.clusterName, "name", "", "Name of the cluster to import")

	o.applierScenariosOptions.AddFlags(cmd.Flags())
	o.applierScenariosOptions.ConfigFlags.AddFlags(cmd.Flags())

	return cmd
}
