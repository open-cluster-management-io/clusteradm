// Copyright Contributors to the Open Cluster Management project
package verbs

import (
	"fmt"

	appliercmd "github.com/open-cluster-management/applier/pkg/applier/cmd"
	attachcluster "github.com/open-cluster-management/cm-cli/pkg/cmd/attach/cluster"
	createcluster "github.com/open-cluster-management/cm-cli/pkg/cmd/create/cluster"
	deletecluster "github.com/open-cluster-management/cm-cli/pkg/cmd/delete/cluster"
	detachcluster "github.com/open-cluster-management/cm-cli/pkg/cmd/detach/cluster"
	getclusters "github.com/open-cluster-management/cm-cli/pkg/cmd/get/clusters"
	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"k8s.io/kubectl/pkg/cmd/get"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

//NewVerb creates a new verb
func NewVerb(verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	switch verb {
	case "create":
		return newVerbCreate(verb, f, streams)
	case "get":
		return newVerbGet(verb, f, streams)
	case "update":
		return newVerbUpdate(verb, f, streams)
	case "delete":
		return newVerbDelete(verb, f, streams)
	case "list":
		return newVerbList(verb, f, streams)
	case "attach":
		return newVerbAttach(verb, f, streams)
	case "applier":
		return newVerbApplier(verb, f, streams)
	case "detach":
		return newVerbDetach(verb, f, streams)
	}
	panic(fmt.Sprintf("Unknow verb: %s", verb))
}

func newVerbCreate(verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use: verb,
	}
	cmd.AddCommand(
		createcluster.NewCmd(streams),
	)

	return cmd
}

func newVerbGet(verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := get.NewCmdGet("cm", f, streams)
	cmd.AddCommand(
		getclusters.NewCmd(f, streams),
	)
	return cmd
}

func newVerbUpdate(verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   verb,
		Short: "Not yet implemented",
	}

	return cmd
}

func newVerbDelete(verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use: verb,
	}
	cmd.AddCommand(
		deletecluster.NewCmd(streams),
	)

	return cmd
}

func newVerbList(verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   verb,
		Short: "Not yet implemented",
	}

	return cmd
}

func newVerbApplier(verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := appliercmd.NewCmd(streams)

	return cmd
}

func newVerbAttach(verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   verb,
		Short: "Attach cluster to hub",
	}

	cmd.AddCommand(attachcluster.NewCmd(streams))

	return cmd
}

func newVerbDetach(verb string, f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   verb,
		Short: "Detatch a cluster from the hub",
	}

	cmd.AddCommand(detachcluster.NewCmd(streams))

	return cmd
}
