// Copyright Contributors to the Open Cluster Management project
package verbs

import (
	"github.com/spf13/cobra"
	"open-cluster-management.io/clusteradm/pkg/cmd/version"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	acceptclusters "open-cluster-management.io/clusteradm/pkg/cmd/accept"
	inithub "open-cluster-management.io/clusteradm/pkg/cmd/init"
	joinhub "open-cluster-management.io/clusteradm/pkg/cmd/join"
)

func NewVerbVersion(parent string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := version.NewCmd(f, streams)

	return cmd
}

func NewVerbInit(parent string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	return inithub.NewCmd(f, streams)
}

func NewVerbJoin(parent string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	return joinhub.NewCmd(f, streams)
}

func NewVerbAccept(parent string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	return acceptclusters.NewCmd(f, streams)
}
