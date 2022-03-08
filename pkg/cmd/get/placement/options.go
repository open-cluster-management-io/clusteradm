// Copyright Contributors to the Open Cluster Management project
package placement

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	clusterv1beta1 "open-cluster-management.io/api/client/cluster/clientset/versioned/typed/cluster/v1beta1"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
)

type Options struct {
	//ClusteradmFlags: The generic optiosn from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	Streams         genericclioptions.IOStreams
	Client          *clusterv1beta1.ClusterV1beta1Client
	PlacementName   string
	Namespace       string
	Output          string
	printer         *printer.PrinterOption
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
		Kind:  "Placement",
	},
	ColumnLabels:     []string{},
	SortBy:           "",
	AllowMissingKeys: true,
}
