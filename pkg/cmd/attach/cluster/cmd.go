// Copyright Contributors to the Open Cluster Management project
package cluster

import (
	"fmt"
	"path/filepath"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/attach/cluster/scenario"
	"github.com/open-cluster-management/cm-cli/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
# Attach a cluster
%[1]s attach cluster --values values.yaml

# Attach a cluster with overwritting the cluster name
%[1]s attach cluster --values values.yaml --name mycluster
`

const (
	scenarioDirectory = "attach"
)

var valuesTemplatePath = filepath.Join(scenarioDirectory, "values-template.yaml")

// NewCmd provides a cobra command wrapping NewCmdImportCluster
func NewCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(streams)

	cmd := &cobra.Command{
		Use:          "cluster",
		Short:        "Import a cluster",
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
	cmd.Flags().StringVar(&o.clusterServer, "cluster-server", "", "cluster server url of the cluster to import")
	cmd.Flags().StringVar(&o.clusterToken, "cluster-token", "", "token to access the cluster to import")
	cmd.Flags().StringVar(&o.clusterKubeConfig, "cluster-kubeconfigr", "", "path to the kubeconfig the cluster to import")
	cmd.Flags().StringVar(&o.importFile, "import-file", "", "the file which will contain the import secret for manual import")

	o.applierScenariosOptions.AddFlags(cmd.Flags())
	o.applierScenariosOptions.ConfigFlags.AddFlags(cmd.Flags())

	return cmd
}
