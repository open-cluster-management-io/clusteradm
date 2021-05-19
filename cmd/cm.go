// Copyright Contributors to the Open Cluster Management project

package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/verbs"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	cliflag "k8s.io/component-base/cli/flag"
	cmdconfig "k8s.io/kubectl/pkg/cmd/config"
	"k8s.io/kubectl/pkg/cmd/options"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func main() {
	streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

	// flags := pflag.NewFlagSet("cm", pflag.ExitOnError)
	// pflag.CommandLine = flags

	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(configFlags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	root := newCmdCMVerbs(f, streams)

	flags := root.PersistentFlags()
	matchVersionKubeConfigFlags.AddFlags(flags)
	flags.SetNormalizeFunc(cliflag.WarnWordSepNormalizeFunc) // Warn for "_" flags

	// Normalize all flags that are coming from other packages or pre-configurations
	// a.k.a. change all "_" to "-". e.g. glog package
	flags.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	// From this point and forward we get warnings on flags that contain "_" separators
	root.SetGlobalNormalizationFunc(cliflag.WarnWordSepNormalizeFunc)

	configFlags.AddFlags(flags)
	root.AddCommand(cmdconfig.NewCmdConfig(f, clientcmd.NewDefaultPathOptions(), streams))
	// root.AddCommand(plugin.NewCmdPlugin(f, streams))
	// root.AddCommand(version.NewCmdVersion(f, streams))
	// root.AddCommand(apiresources.NewCmdAPIVersions(f, streams))
	// root.AddCommand(apiresources.NewCmdAPIResources(f, streams))
	root.AddCommand(options.NewCmdOptions(streams.Out))

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// NewCmdNamespace provides a cobra command wrapping NamespaceOptions
func newCmdCMVerbs(f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{Use: "cm"}
	cmd.AddCommand(
		verbs.NewVerb("create", f, streams),
		verbs.NewVerb("get", f, streams),
		// verbs.NewVerb("update", streams),
		verbs.NewVerb("delete", f, streams),
		// verbs.NewVerb("list", streams),
		verbs.NewVerb("applier", f, streams),
		verbs.NewVerb("attach", f, streams),
		verbs.NewVerb("detach", f, streams),
	)

	return cmd
}
