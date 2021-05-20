// Copyright Contributors to the Open Cluster Management project
package verbs

import (
	"fmt"

	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"k8s.io/kubectl/pkg/cmd/get"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

//NewVerb creates a new verb
func NewVerb(parent string, verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	switch verb {
	case "get":
		return newVerbGet(parent, verb, f, streams)
	}
	panic(fmt.Sprintf("Unknow verb: %s", verb))
}

func newVerbGet(parent string, verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := get.NewCmdGet(parent, f, streams)
	return cmd
}
