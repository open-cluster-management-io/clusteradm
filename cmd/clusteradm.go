// Copyright Contributors to the Open Cluster Management project

package main

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	cliflag "k8s.io/component-base/cli/flag"
	cmdconfig "k8s.io/kubectl/pkg/cmd/config"
	"k8s.io/kubectl/pkg/cmd/options"
	"k8s.io/kubectl/pkg/cmd/plugin"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"open-cluster-management.io/clusteradm/pkg/cmd/verbs"
)

func main() {
	streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(configFlags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	root := newCmdVerbs("clusteradm", f, streams)

	flags := root.PersistentFlags()
	matchVersionKubeConfigFlags.AddFlags(flags)
	flags.SetNormalizeFunc(cliflag.WarnWordSepNormalizeFunc) // Warn for "_" flags

	flags.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	// From this point and forward we get warnings on flags that contain "_" separators
	root.SetGlobalNormalizationFunc(cliflag.WarnWordSepNormalizeFunc)

	configFlags.AddFlags(flags)
	root.AddCommand(cmdconfig.NewCmdConfig(f, clientcmd.NewDefaultPathOptions(), streams))
	root.AddCommand(options.NewCmdOptions(streams.Out))
	//enable plugin functionality: all `os.Args[0]-<binary>` in the $PATH will be available for plugin
	plugin.ValidPluginFilenamePrefixes = []string{os.Args[0]}
	root.AddCommand(plugin.NewCmdPlugin(f, streams))

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// NewCmdNamespace provides a cobra command wrapping NamespaceOptions
func newCmdVerbs(parent string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{Use: parent}
	cmd.AddCommand(
		verbs.NewVerbVersion("version", f, streams),
		verbs.NewVerbInit("init", f, streams),
		verbs.NewVerbJoin("join", f, streams),
		verbs.NewVerbAccept("accept", f, streams),
	)

	return cmd
}
