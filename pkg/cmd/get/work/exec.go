// Copyright Contributors to the Open Cluster Management project
package work

import (
	"context"
	"fmt"

	"github.com/disiqueira/gotree"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/printers"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	workclient "open-cluster-management.io/api/client/work/clientset/versioned"
	workapiv1 "open-cluster-management.io/api/work/v1"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
)

const (
	workGroup      = "work.open-cluster-management.io"
	workAPIVersion = "v1"
	workKind       = "ManifestWork"
)

var option = printers.PrintOptions{
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
}

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {

	if len(args) > 1 {
		return fmt.Errorf("can only specify one manifestwork")
	}

	if len(args) == 1 {
		o.workName = args[0]
	}

	o.printer = printer.NewPrinter(o.ClusteradmFlags.Output)

	return nil
}

func (o *Options) validate() (err error) {
	if len(o.cluster) == 0 {
		return fmt.Errorf("cluster name must be specified")
	}
	return nil
}

func (o *Options) run() (err error) {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	clusterClient, err := clusterclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	workClient, err := workclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	_, err = clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), o.cluster, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if o.printer.IsYaml() {
		return o.printWorkYaml(workClient)
	}
	if o.printer.IsTree() {
		return o.printWorkTree(workClient)
	}
	return o.printWorkTable(workClient)
}

func (o *Options) printWorkTree(workClient *workclient.Clientset) error {
	root := gotree.New("<ManifestWork>")

	if len(o.workName) == 0 {
		workList, err := workClient.WorkV1().ManifestWorks(o.cluster).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, item := range workList.Items {
			printer.PrintWorkDetail(root, &item)
		}
	} else {
		work, err := workClient.WorkV1().ManifestWorks(o.cluster).Get(context.TODO(), o.workName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		printer.PrintWorkDetail(root, work)
	}

	return o.printer.PrintObject(o.Streams.Out, root, printers.PrintOptions{})
}

func (o *Options) printWorkYaml(workClient *workclient.Clientset) error {
	if len(o.workName) == 0 {
		workList, err := workClient.WorkV1().ManifestWorks(o.cluster).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, item := range workList.Items {
			err = o.printer.PrintObject(o.Streams.Out, &item, printers.PrintOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	}

	work, err := workClient.WorkV1().ManifestWorks(o.cluster).Get(context.TODO(), o.workName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	work.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   workGroup,
		Version: workAPIVersion,
		Kind:    workKind,
	})

	return o.printer.PrintObject(o.Streams.Out, work, printers.PrintOptions{})
}

func (o *Options) printWorkTable(workClient *workclient.Clientset) error {
	if len(o.workName) == 0 {
		workList, err := workClient.WorkV1().ManifestWorks(o.cluster).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}

		table := converToTable(workList)
		return o.printer.PrintObject(o.Streams.Out, table, option)
	}

	work, err := workClient.WorkV1().ManifestWorks(o.cluster).Get(context.TODO(), o.workName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	list := workapiv1.ManifestWorkList{Items: []workapiv1.ManifestWork{*work}}
	table := converToTable(&list)
	return o.printer.PrintObject(o.Streams.Out, table, option)
}

func converToTable(works *workapiv1.ManifestWorkList) *metav1.Table {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Cluster", Type: "string"},
			{Name: "Number Of Manifests", Type: "integer"},
			{Name: "Applied", Type: "string"},
			{Name: "Available", Type: "string"},
		},
		Rows: []metav1.TableRow{},
	}

	for _, work := range works.Items {
		row := convertRow(work)
		table.Rows = append(table.Rows, row)
	}

	return table
}

func convertRow(work workapiv1.ManifestWork) metav1.TableRow {
	var applied, available string

	appliedCond := meta.FindStatusCondition(work.Status.Conditions, workapiv1.WorkApplied)
	if appliedCond != nil {
		applied = string(appliedCond.Status)
	}

	availableCond := meta.FindStatusCondition(work.Status.Conditions, workapiv1.WorkAvailable)
	if availableCond != nil {
		available = string(availableCond.Status)
	}

	return metav1.TableRow{
		Cells:  []interface{}{work.Name, work.Namespace, len(work.Spec.Workload.Manifests), applied, available},
		Object: runtime.RawExtension{Object: &work},
	}
}
