// Copyright Contributors to the Open Cluster Management project

package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/verbs"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {
	flags := pflag.NewFlagSet("cm", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := newCmdCMVerbs(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// NewCmdNamespace provides a cobra command wrapping NamespaceOptions
func newCmdCMVerbs(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{Use: "cm"}
	cmd.AddCommand(
		verbs.NewVerb("create", streams),
		// verbs.NewVerb("get", streams),
		// verbs.NewVerb("update", streams),
		verbs.NewVerb("delete", streams),
		// verbs.NewVerb("list", streams),
		verbs.NewVerb("applier", streams),
		// verbs.NewVerb("attach", streams),
		// verbs.NewVerb("detach", streams),
	)

	return cmd
}
