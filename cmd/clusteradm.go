// Copyright Contributors to the Open Cluster Management project

package main

import (
	"flag"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	cliflag "k8s.io/component-base/cli/flag"
	cmdconfig "k8s.io/kubectl/pkg/cmd/config"
	"k8s.io/kubectl/pkg/cmd/options"
	"k8s.io/kubectl/pkg/cmd/plugin"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"open-cluster-management.io/clusteradm/pkg/cmd/version"

	acceptclusters "open-cluster-management.io/clusteradm/pkg/cmd/accept"
	deletecmd "open-cluster-management.io/clusteradm/pkg/cmd/delete"
	"open-cluster-management.io/clusteradm/pkg/cmd/get"
	inithub "open-cluster-management.io/clusteradm/pkg/cmd/init"
	joinhub "open-cluster-management.io/clusteradm/pkg/cmd/join"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

func main() {
	streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(configFlags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)
	clusteradmFlags := genericclioptionsclusteradm.NewClusteradmFlags(f)

	root :=
		&cobra.Command{
			Use: "clusteradm",
		}

	flags := root.PersistentFlags()
	matchVersionKubeConfigFlags.AddFlags(flags)
	flags.SetNormalizeFunc(cliflag.WarnWordSepNormalizeFunc) // Warn for "_" flags

	flags.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	// From this point and forward we get warnings on flags that contain "_" separators
	root.SetGlobalNormalizationFunc(cliflag.WarnWordSepNormalizeFunc)

	configFlags.AddFlags(flags)
	clusteradmFlags.AddFlags(flags)
	flags.AddGoFlagSet(flag.CommandLine)

	root.AddCommand(cmdconfig.NewCmdConfig(f, clientcmd.NewDefaultPathOptions(), streams))
	root.AddCommand(options.NewCmdOptions(streams.Out))
	//enable plugin functionality: all `os.Args[0]-<binary>` in the $PATH will be available for plugin
	plugin.ValidPluginFilenamePrefixes = []string{os.Args[0]}
	root.AddCommand(plugin.NewCmdPlugin(f, streams))

	groups := ktemplates.CommandGroups{
		{
			Message: "General commands:",
			Commands: []*cobra.Command{
				version.NewCmd(clusteradmFlags, streams),
			},
		},
		{
			Message: "Registration commands:",
			Commands: []*cobra.Command{
				get.NewCmd(clusteradmFlags, streams),
				deletecmd.NewCmd(clusteradmFlags, streams),
				inithub.NewCmd(clusteradmFlags, streams),
				joinhub.NewCmd(clusteradmFlags, streams),
				acceptclusters.NewCmd(clusteradmFlags, streams),
			},
		},
	}
	groups.Add(root)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
