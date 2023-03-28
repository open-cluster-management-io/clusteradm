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
	utilpointer "k8s.io/utils/pointer"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"

	// commands
	acceptclusters "open-cluster-management.io/clusteradm/pkg/cmd/accept"
	addon "open-cluster-management.io/clusteradm/pkg/cmd/addon"
	clean "open-cluster-management.io/clusteradm/pkg/cmd/clean"
	"open-cluster-management.io/clusteradm/pkg/cmd/clusterset"
	"open-cluster-management.io/clusteradm/pkg/cmd/create"
	deletecmd "open-cluster-management.io/clusteradm/pkg/cmd/delete"
	"open-cluster-management.io/clusteradm/pkg/cmd/get"
	inithub "open-cluster-management.io/clusteradm/pkg/cmd/init"
	install "open-cluster-management.io/clusteradm/pkg/cmd/install"
	joinhub "open-cluster-management.io/clusteradm/pkg/cmd/join"
	"open-cluster-management.io/clusteradm/pkg/cmd/proxy"
	unjoin "open-cluster-management.io/clusteradm/pkg/cmd/unjoin"
	"open-cluster-management.io/clusteradm/pkg/cmd/upgrade"
	"open-cluster-management.io/clusteradm/pkg/cmd/version"
)

func main() {
	root :=
		&cobra.Command{
			Use: "clusteradm",
			Long: ktemplates.LongDesc(`
			clusteradm controls the OCM control plane.
			
			Find more information at:
				https://github.com/open-cluster-management-io/clusteradm/blob/main/README.md
			`),
			Run: runHelp,
		}

	flags := root.PersistentFlags()
	flags.SetNormalizeFunc(cliflag.WarnWordSepNormalizeFunc) // Warn for "_" flags
	flags.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)

	//kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeConfigFlags := &genericclioptions.ConfigFlags{
		KubeConfig: utilpointer.String(""),
		Context:    utilpointer.String(""),
	}
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
	clusteradmFlags.SetContext(kubeConfigFlags.Context)

	// From this point and forward we get warnings on flags that contain "_" separators

	root.AddCommand(cmdconfig.NewCmdConfig(clientcmd.NewDefaultPathOptions(), streams))
	root.AddCommand(options.NewCmdOptions(streams.Out))
	//addon plugin functionality: all `os.Args[0]-<binary>` in the $PATH will be available for plugin
	plugin.ValidPluginFilenamePrefixes = []string{os.Args[0]}
	root.AddCommand(plugin.NewCmdPlugin(streams))

	groups := ktemplates.CommandGroups{
		{
			Message: "General commands:",
			Commands: []*cobra.Command{
				create.NewCmd(clusteradmFlags, streams),
				deletecmd.NewCmd(clusteradmFlags, streams),
				get.NewCmd(clusteradmFlags, streams),
				install.NewCmd(clusteradmFlags, streams),
				upgrade.NewCmd(clusteradmFlags, streams),
				version.NewCmd(clusteradmFlags, streams),
			},
		},
		{
			Message: "Registration commands:",
			Commands: []*cobra.Command{
				acceptclusters.NewCmd(clusteradmFlags, streams),
				clean.NewCmd(clusteradmFlags, streams),
				inithub.NewCmd(clusteradmFlags, streams),
				joinhub.NewCmd(clusteradmFlags, streams),
				unjoin.NewCmd(clusteradmFlags, streams),
			},
		},
		{
			Message: "Cluster Management commands:",
			Commands: []*cobra.Command{
				addon.NewCmd(clusteradmFlags, streams),
				clusterset.NewCmd(clusteradmFlags, streams),
				proxy.NewCmd(clusteradmFlags, streams),
			},
		},
	}
	groups.Add(root)

	filters := []string{"options"}

	ktemplates.ActsAsRootCommand(root, filters, groups...)

	err := root.Execute()
	if err != nil {
		klog.V(1).ErrorS(err, "Error:")
	}
	klog.Flush()
	if err != nil {
		os.Exit(1)
	}
}

func runHelp(cmd *cobra.Command, args []string) {
	_ = cmd.Help()
}
