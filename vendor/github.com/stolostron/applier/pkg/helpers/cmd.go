// Copyright Red Hat

package helpers

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	genericclioptionsapplier "github.com/stolostron/applier/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/options"
	"k8s.io/kubectl/pkg/cmd/plugin"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/stolostron/applier/pkg/asset"
)

func NewRootCmd() (*cobra.Command, *genericclioptionsapplier.ApplierFlags, genericclioptions.IOStreams) {
	root := &cobra.Command{
		Use:   "applier",
		Short: "apply templated resources",
		//This remove the auto-generated tag in the cobra doc
		DisableAutoGenTag: true,
	}

	flags := root.PersistentFlags()
	flags.SetNormalizeFunc(cliflag.WarnWordSepNormalizeFunc) // Warn for "_" flags

	// Normalize all flags that are coming from other packages or pre-configurations
	// a.k.a. change all "_" to "-". e.g. glog package
	flags.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)

	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeConfigFlags.WrapConfigFn = setQPS
	kubeConfigFlags.AddFlags(flags)
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)

	matchVersionKubeConfigFlags.AddFlags(flags)

	klog.InitFlags(nil)
	flags.AddGoFlagSet(flag.CommandLine)

	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)
	// From this point and forward we get warnings on flags that contain "_" separators
	root.SetGlobalNormalizationFunc(cliflag.WarnWordSepNormalizeFunc)
	streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

	applierFlags := genericclioptionsapplier.NewApplierFlags(f)
	applierFlags.AddFlags(flags)

	// root.AddCommand(cmdconfig.NewCmdConfig(f, clientcmd.NewDefaultPathOptions(), streams))
	root.AddCommand(options.NewCmdOptions(streams.Out))

	//enable plugin functionality: all `os.Args[0]-<binary>` in the $PATH will be available for plugin
	plugin.ValidPluginFilenamePrefixes = []string{os.Args[0]}
	root.AddCommand(plugin.NewCmdPlugin(streams))
	return root, applierFlags, streams
}

func setQPS(r *rest.Config) *rest.Config {
	r.QPS = QPS
	r.Burst = Burst
	return r
}

func GetExampleHeader() string {
	switch os.Args[0] {
	case "oc":
		return "oc cm"
	case "kubectl":
		return "kubectl cm"
	default:
		return os.Args[0]
	}
}

func UsageTempate(cmd *cobra.Command, reader asset.ScenarioReader, valuesTemplatePath string) string {
	baseUsage := cmd.UsageTemplate()
	b, err := reader.Asset(valuesTemplatePath)
	if err != nil {
		return fmt.Sprintf("%s\n\n Values template:\n%s", baseUsage, err.Error())
	}
	return fmt.Sprintf("%s\n\n Values template:\n%s", baseUsage, string(b))
}

func DryRunMessage(dryRun bool) {
	if dryRun {
		fmt.Printf("%s is running in dry-run mode\n", GetExampleHeader())
	}
}
