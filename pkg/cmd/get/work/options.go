// Copyright Contributors to the Open Cluster Management project
package work

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags
	//A list of comma separated cluster names
	cluster string

	workName string

	Streams genericclioptions.IOStreams

	printer printers.ResourcePrinter
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
		printer: printers.NewTablePrinter(printers.PrintOptions{
			NoHeaders:     false,
			WithNamespace: false,
			WithKind:      false,
			Wide:          false,
			ShowLabels:    false,
			Kind: schema.GroupKind{
				Group: "work.open-cluster-management.io",
				Kind:  "ManifestWork",
			},
			ColumnLabels:     []string{},
			SortBy:           "",
			AllowMissingKeys: true,
		}),
	}
}
