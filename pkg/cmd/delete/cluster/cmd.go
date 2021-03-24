// Copyright Contributors to the Open Cluster Management project
package cluster

import (
	"fmt"
	"path/filepath"

	"github.com/open-cluster-management/cm-cli/pkg/helpers"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
# Delete a cluster
%[1]s delete cluster --values values.yaml

# Delete a cluster with overwritting the cluster name
%[1]s delete cluster --values values.yaml --name mycluster
`

const (
	scenarioDirectory = "scenarios/delete"
)

var valuesTemplatePath = filepath.Join(scenarioDirectory, "values-template.yaml")

// NewCmd ...
func NewCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(streams)

	cmd := &cobra.Command{
		Use:          "cluster",
		Short:        "Delete a cluster",
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

	cmd.SetUsageTemplate(applierscenarios.UsageTempate(cmd, valuesTemplatePath))
	cmd.Flags().StringVar(&o.clusterName, "name", "", "Name of the cluster to import")

	o.applierScenariosOptions.AddFlags(cmd.Flags())
	o.applierScenariosOptions.ConfigFlags.AddFlags(cmd.Flags())

	return cmd
}
