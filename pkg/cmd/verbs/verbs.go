// Copyright Contributors to the Open Cluster Management project
package verbs

import (
	"github.com/spf13/cobra"
	"open-cluster-management.io/clusteradm/pkg/cmd/version"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func NewVerbVersion(verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := version.NewCmd(f, streams)

	return cmd
}
