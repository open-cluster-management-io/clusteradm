// Copyright Contributors to the Open Cluster Management project

package main

import (
	"flag"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/tools/clientcmd"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"

	cmdconfig "k8s.io/kubectl/pkg/cmd/config"
	"k8s.io/kubectl/pkg/cmd/options"
	"k8s.io/kubectl/pkg/cmd/plugin"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"open-cluster-management.io/clusteradm/pkg/cmd/version"

	acceptclusters "open-cluster-management.io/clusteradm/pkg/cmd/accept"
	clean "open-cluster-management.io/clusteradm/pkg/cmd/clean"
	deletecmd "open-cluster-management.io/clusteradm/pkg/cmd/delete"
	enable "open-cluster-management.io/clusteradm/pkg/cmd/enable"
	"open-cluster-management.io/clusteradm/pkg/cmd/get"
	inithub "open-cluster-management.io/clusteradm/pkg/cmd/init"
	install "open-cluster-management.io/clusteradm/pkg/cmd/install"
	joinhub "open-cluster-management.io/clusteradm/pkg/cmd/join"
	unjoin "open-cluster-management.io/clusteradm/pkg/cmd/unjoin"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

func main() {
	root :=
		&cobra.Command{
			Use: "clusteradm",
		}

	flags := root.PersistentFlags()
	flags.SetNormalizeFunc(cliflag.WarnWordSepNormalizeFunc) // Warn for "_" flags
	flags.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)

	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeConfigFlags.AddFlags(flags)
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	matchVersionKubeConfigFlags.AddFlags(flags)

	klog.InitFlags(nil)
	flags.AddGoFlagSet(flag.CommandLine)

	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)
	root.SetGlobalNormalizationFunc(cliflag.WarnWordSepNormalizeFunc)
	streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

	clusteradmFlags := genericclioptionsclusteradm.NewClusteradmFlags(f)
	clusteradmFlags.AddFlags(flags)

	// From this point and forward we get warnings on flags that contain "_" separators

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
				clean.NewCmd(clusteradmFlags, streams),
				inithub.NewCmd(clusteradmFlags, streams),
				joinhub.NewCmd(clusteradmFlags, streams),
				unjoin.NewCmd(clusteradmFlags, streams),
				enable.NewCmd(clusteradmFlags, streams),
				install.NewCmd(clusteradmFlags, streams),
				acceptclusters.NewCmd(clusteradmFlags, streams),
			},
		},
	}
	groups.Add(root)
	err := root.Execute()
	if err != nil {
		klog.V(1).ErrorS(err, "Error:")
	}
	klog.Flush()
	if err != nil {
		os.Exit(1)
	}
}
