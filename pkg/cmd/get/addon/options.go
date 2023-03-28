// Copyright Contributors to the Open Cluster Management project
package addon

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	ClusterOptions  *genericclioptionsclusteradm.ClusterOption
	// A list of addon name to show
	addons []string

	Streams genericclioptions.IOStreams

	printer *printer.PrinterOption
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		ClusterOptions:  genericclioptionsclusteradm.NewClusterOption().AllowUnset(),
		Streams:         streams,
		printer:         printer.NewPrinterOption(pntOpt),
	}
}

var pntOpt = printers.PrintOptions{
	NoHeaders:     false,
	WithNamespace: false,
	WithKind:      false,
	Wide:          false,
	ShowLabels:    false,
	Kind: schema.GroupKind{
		Group: "add.open-cluster-management.io",
		Kind:  "ClusterManagementAddon",
	},
	ColumnLabels:     []string{},
	SortBy:           "",
	AllowMissingKeys: true,
}
