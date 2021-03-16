// Copyright Contributors to the Open Cluster Management project
package verbs

import (
	"fmt"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/apply"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/attach"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/create"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/delete"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/detach"
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
		return newVerbApply(verb, streams)
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
		create.NewCmdCreateCluster(streams),
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
		delete.NewCmdDeleteCluster(streams),
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

func newVerbApply(verb string, streams genericclioptions.IOStreams) *cobra.Command {
	o := apply.NewApplierOptions(streams)

	cmd := &cobra.Command{
		Use:          verb,
		Short:        "apply templates",
		Example:      fmt.Sprintf(apply.ApplyExample, "kubectl"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	o.AddFlags(cmd.Flags())
	o.ConfigFlags.AddFlags(cmd.Flags())

	return cmd
}

func newVerbAttach(verb string, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   verb,
		Short: "Attach cluster to hub",
	}

	cmd.AddCommand(attach.NewCmdAttachCluster(streams))

	return cmd
}

func newVerbDetach(verb string, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   verb,
		Short: "Detatch a cluster from the hub",
	}

	cmd.AddCommand(detach.NewCmdDetachCluster(streams))

	return cmd
}
