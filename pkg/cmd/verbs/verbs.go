// Copyright Contributors to the Open Cluster Management project
package verbs

import (
	"fmt"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/apply"
	attachcluster "github.com/open-cluster-management/cm-cli/pkg/cmd/attach/cluster"
	createcluster "github.com/open-cluster-management/cm-cli/pkg/cmd/create/cluster"
	deletecluster "github.com/open-cluster-management/cm-cli/pkg/cmd/delete/cluster"
	detachcluster "github.com/open-cluster-management/cm-cli/pkg/cmd/detach/cluster"
	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

//NewVerb creates a new verb
func NewVerb(verb string, streams genericclioptions.IOStreams) *cobra.Command {
	switch verb {
	case "create":
		return newVerbCreate(verb, streams)
	case "get":
		return newVerbGet(verb, streams)
	case "update":
		return newVerbUpdate(verb, streams)
	case "delete":
		return newVerbDelete(verb, streams)
	case "list":
		return newVerbList(verb, streams)
	case "attach":
		return newVerbAttach(verb, streams)
	case "applier":
		return newVerbApplier(verb, streams)
	case "detach":
		return newVerbDetach(verb, streams)
	}
	panic(fmt.Sprintf("Unknow verb: %s", verb))
}

func newVerbCreate(verb string, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use: verb,
	}
	cmd.AddCommand(
		createcluster.NewCmd(streams),
	)

	return cmd
}

func newVerbGet(verb string, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   verb,
		Short: "Not yet implemented",
	}

	return cmd
}

func newVerbUpdate(verb string, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   verb,
		Short: "Not yet implemented",
	}

	return cmd
}

func newVerbDelete(verb string, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use: verb,
	}
	cmd.AddCommand(
		deletecluster.NewCmd(streams),
	)

	return cmd
}

func newVerbList(verb string, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   verb,
		Short: "Not yet implemented",
	}

	return cmd
}

func newVerbApplier(verb string, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := apply.NewCmd(streams)

	return cmd
}

func newVerbAttach(verb string, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   verb,
		Short: "Attach cluster to hub",
	}

	cmd.AddCommand(attachcluster.NewCmd(streams))

	return cmd
}

func newVerbDetach(verb string, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   verb,
		Short: "Detatch a cluster from the hub",
	}

	cmd.AddCommand(detachcluster.NewCmd(streams))

	return cmd
}
