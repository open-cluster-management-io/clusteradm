// Copyright Contributors to the Open Cluster Management project
package verbs

import (
	"github.com/spf13/cobra"
	"open-cluster-management.io/clusteradm/pkg/cmd/version"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	inithub "open-cluster-management.io/clusteradm/pkg/cmd/init/hub"
)

func NewVerbVersion(parent string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := version.NewCmd(f, streams)

	return cmd
}

func NewVerbInit(parent string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   parent,
		Short: "Initialize the hub",
	}

	cmd.AddCommand(inithub.NewCmd(f, streams))

	return cmd
}
