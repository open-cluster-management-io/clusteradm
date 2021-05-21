// Copyright Contributors to the Open Cluster Management project
package verbs

import (
	"github.com/spf13/cobra"
	"open-cluster-management.io/clusteradm/pkg/cmd/version"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"k8s.io/kubectl/pkg/cmd/get"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func NewVerbGet(verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := get.NewCmdGet(verb, f, streams)
	return cmd
}

func NewVerbVersion(verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := version.NewCmd(streams)

	return cmd
}
