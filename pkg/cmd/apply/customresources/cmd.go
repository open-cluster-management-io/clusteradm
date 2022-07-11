// Copyright Contributors to the Open Cluster Management project
package customresources

import (
	"fmt"

	"open-cluster-management.io/clusteradm/pkg/cmd/apply/common"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
# Apply custom-resources templates
%[1]s apply custom-resources --values values.yaml --path template_path1 tempalte_path2...
`

// NewCmd ...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := common.NewOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "custom-resources",
		Short:        "apply custom-resources templates located in paths",
		Long:         "apply custom-resources templates located in paths with a values.yaml, the list of path can be a path to a file or a directory",
		Example:      fmt.Sprintf(example, helpers.GetExampleHeader()),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			o.ResourcesType = common.CustomResources
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

	cmd.Flags().StringVar(&o.ValuesPath, "values", "", "The files containing the values")
	cmd.Flags().StringArrayVar(&o.Paths, "paths", []string{}, "The list of template paths")
	cmd.Flags().StringVar(&o.OutputFile, "output-file", "", "The generated resources will be copied in the specified file")
	return cmd
}
