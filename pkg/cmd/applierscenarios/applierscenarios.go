// Copyright Contributors to the Open Cluster Management project
package applierscenarios

import (
	"fmt"

	"github.com/open-cluster-management/cm-cli/pkg/resources"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var ApplierScenariosExample = `
# Import a cluster
%[1]s --values values.yaml
`

type ApplierScenariosOptions struct {
	ConfigFlags *genericclioptions.ConfigFlags

	OutFile         string
	ValuesPath      string
	Timeout         int
	Force           bool
	Silent          bool
	OutTemplatesDir string

	genericclioptions.IOStreams
}

func NewApplierScenariosOptions(streams genericclioptions.IOStreams) *ApplierScenariosOptions {
	return &ApplierScenariosOptions{
		ConfigFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

func (o *ApplierScenariosOptions) AddFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&o.OutFile, "outfile", "o", "",
		"Output file. If set nothing will be applied but a file will be generate "+
			"which you can apply later with 'kubectl <create|apply|delete> -f")
	flagSet.StringVar(&o.ValuesPath, "values", "", "The files containing the values")
	flagSet.IntVar(&o.Timeout, "t", 5, "Timeout in second to apply one resource, default 5 sec")
	flagSet.BoolVarP(&o.Force, "force", "f", false, "If set, the finalizers will be removed before delete")
	flagSet.BoolVar(&o.Silent, "silent", false, "If set the applier will run silently")
	flagSet.StringVar(&o.OutTemplatesDir, "out-templates-dir", "", "The templates will be saved in the provided directory")
}

func UsageTempate(cmd *cobra.Command, valuesTemplatePath string) string {
	baseUsage := cmd.UsageTemplate()
	b, err := resources.NewResourcesReader().Asset(valuesTemplatePath)
	if err != nil {
		return fmt.Sprintf("%s\n\n Values template:\n%s", baseUsage, err.Error())
	}
	return fmt.Sprintf("%s\n\n Values template:\n%s", baseUsage, string(b))
}
