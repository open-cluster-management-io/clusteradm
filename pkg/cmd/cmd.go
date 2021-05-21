// Copyright Contributors to the Open Cluster Management project

package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	cliflag "k8s.io/component-base/cli/flag"
	cmdconfig "k8s.io/kubectl/pkg/cmd/config"
	"k8s.io/kubectl/pkg/cmd/options"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"open-cluster-management.io/clusteradm/pkg/cmd/verbs"
)

func NewRootCmd(parent string, streams genericclioptions.IOStreams) (*cobra.Command, cmdutil.Factory) {

	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(configFlags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	root := newCmdVerbs(parent, f, streams)

	flags := root.PersistentFlags()
	matchVersionKubeConfigFlags.AddFlags(flags)
	flags.SetNormalizeFunc(cliflag.WarnWordSepNormalizeFunc) // Warn for "_" flags

	flags.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	// From this point and forward we get warnings on flags that contain "_" separators
	root.SetGlobalNormalizationFunc(cliflag.WarnWordSepNormalizeFunc)

	configFlags.AddFlags(flags)
	root.AddCommand(cmdconfig.NewCmdConfig(f, clientcmd.NewDefaultPathOptions(), streams))
	root.AddCommand(options.NewCmdOptions(streams.Out))
	return root, f
}

// NewCmdNamespace provides a cobra command wrapping NamespaceOptions
func newCmdVerbs(parent string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{Use: parent}
	cmd.AddCommand(
		verbs.NewVerb(parent, "get", f, streams),
	)

	return cmd
}
