// Copyright Contributors to the Open Cluster Management project
package join

import (
	"fmt"
	"path/filepath"

	"open-cluster-management.io/clusteradm/pkg/cmd/join/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var example = `
# Init hub
%[1]s join hub
`

const (
	scenarioDirectory = "join"
)

var valuesTemplatePath = filepath.Join(scenarioDirectory, "values-template.yaml")

// NewCmd ...
func NewCmd(f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(f, streams)

	cmd := &cobra.Command{
		Use:          "join",
		Short:        "join a hub",
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
	cmd.Flags().StringVar(&o.token, "hub-token", "", "The token to access the hub")
	cmd.Flags().StringVar(&o.hubServerExternal, "hub-server", "", "The external api server url to the hub")
	cmd.Flags().StringVar(&o.hubServerInternal, "hub-server-internal", "", "The internal api server url to the hub")
	cmd.Flags().StringVar(&o.clusterName, "name", "", "The name of the joining cluster")
	return cmd
}
