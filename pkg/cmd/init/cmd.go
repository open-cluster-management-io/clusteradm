// Copyright Contributors to the Open Cluster Management project
package init

import (
	"fmt"
	"path/filepath"

	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var example = `
# Init hub
%[1]s init hub
`

const (
	scenarioDirectory = "init"
)

var valuesTemplatePath = filepath.Join(scenarioDirectory, "values-template.yaml")

// NewCmd ...
func NewCmd(f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(f, streams)

	cmd := &cobra.Command{
		Use:          "init",
		Short:        "init hub",
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

	cmd.SetUsageTemplate(helpers.UsageTempate(cmd, scenario.GetScenarioResourcesReader(), valuesTemplatePath))

	return cmd
}
