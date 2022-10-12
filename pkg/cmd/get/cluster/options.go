// Copyright Contributors to the Open Cluster Management project
package cluster

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
	//Clusterset of the clusters
	Clusterset string

	Streams genericclioptions.IOStreams

	printer *printer.PrinterOption
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
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
		Group: "cluster.open-cluster-management.io",
		Kind:  "ManagedCluster",
	},
	ColumnLabels:     []string{},
	SortBy:           "",
	AllowMissingKeys: true,
}
