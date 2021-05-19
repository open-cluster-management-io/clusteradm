// Copyright Contributors to the Open Cluster Management project
package clusters

import (
	"fmt"

	"k8s.io/kubectl/pkg/cmd/get"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/open-cluster-management/cm-cli/pkg/helpers"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	example = `
	# get a cluster
	%[1]s clusters <cluster-name> -oyaml`
)

// NewCmd ...
func NewCmd(f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := get.NewGetOptions("cm", streams)
	cmd := &cobra.Command{
		Use:                   "clusters [(-o|--output=)json|yaml|wide|custom-columns=...|custom-columns-file=...|go-template=...|go-template-file=...|jsonpath=...|jsonpath-file=...] (TYPE[.VERSION][.GROUP] [NAME | -l label] | TYPE[.VERSION][.GROUP]/NAME ...) [flags]",
		Aliases:               []string{"cluster"},
		DisableFlagsInUseLine: true,
		Short:                 "Display one or many resources",
		Example:               fmt.Sprintf(example, helpers.GetExampleHeader()),
		Run: func(cmd *cobra.Command, args []string) {
			args = append([]string{"managedcluster"}, args...)
			cmdutil.CheckErr(o.Complete(f, cmd, args))
			cmdutil.CheckErr(o.Validate(cmd))
			cmdutil.CheckErr(o.Run(f, cmd, args))
		},
		SuggestFor: []string{"list", "ps"},
	}

	o.PrintFlags.AddFlags(cmd)

	cmd.Flags().BoolVarP(&o.Watch, "watch", "w", o.Watch, "After listing/getting the requested object, watch for changes. Uninitialized objects are excluded if no object name is provided.")
	cmd.Flags().BoolVar(&o.WatchOnly, "watch-only", o.WatchOnly, "Watch for changes to the requested object(s), without listing/getting first.")
	cmd.Flags().BoolVar(&o.OutputWatchEvents, "output-watch-events", o.OutputWatchEvents, "Output watch event objects when --watch or --watch-only is used. Existing objects are output as initial ADDED events.")
	cmd.Flags().Int64Var(&o.ChunkSize, "chunk-size", o.ChunkSize, "Return large lists in chunks rather than all at once. Pass 0 to disable. This flag is beta and may change in the future.")
	cmd.Flags().BoolVar(&o.IgnoreNotFound, "ignore-not-found", o.IgnoreNotFound, "If the requested object does not exist the command will return exit code 0.")
	cmd.Flags().StringVarP(&o.LabelSelector, "selector", "l", o.LabelSelector, "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	cmd.Flags().StringVar(&o.FieldSelector, "field-selector", o.FieldSelector, "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	cmd.Flags().BoolVarP(&o.AllNamespaces, "all-namespaces", "A", o.AllNamespaces, "If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
	addOpenAPIPrintColumnFlags(cmd, o)
	addServerPrintColumnFlags(cmd, o)
	cmdutil.AddFilenameOptionFlags(cmd, &o.FilenameOptions, "identifying the resource to get from a server.")
	return cmd
}

const (
	useOpenAPIPrintColumnFlagLabel = "use-openapi-print-columns"
	useServerPrintColumns          = "server-print"
)

func addOpenAPIPrintColumnFlags(cmd *cobra.Command, opt *get.GetOptions) {
	cmd.Flags().BoolVar(&opt.PrintWithOpenAPICols, useOpenAPIPrintColumnFlagLabel, opt.PrintWithOpenAPICols, "If true, use x-kubernetes-print-column metadata (if present) from the OpenAPI schema for displaying a resource.")
	cmd.Flags().MarkDeprecated(useOpenAPIPrintColumnFlagLabel, "deprecated in favor of server-side printing")
}

func addServerPrintColumnFlags(cmd *cobra.Command, opt *get.GetOptions) {
	cmd.Flags().BoolVar(&opt.ServerPrint, useServerPrintColumns, opt.ServerPrint, "If true, have the server return the appropriate table output. Supports extension APIs and CRDs.")
}
