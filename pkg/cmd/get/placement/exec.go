// Copyright Contributors to the Open Cluster Management project
package placement

import (
	"context"
	"fmt"

	"github.com/disiqueira/gotree"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/printers"
	clusterv1alpha1 "open-cluster-management.io/api/client/cluster/clientset/versioned/typed/cluster/v1alpha1"
	"open-cluster-management.io/api/cluster/v1alpha1"
)

const placementLabel = "cluster.open-cluster-management.io/placement"

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	return nil
}

func (o *Options) validate(args []string) (err error) {
	if len(args) > 0 {
		return fmt.Errorf("there should be no argument")
	}

	if o.output != "" && o.output != "tree" && o.output != "table" {
		return fmt.Errorf("output flag should be \"\", \"tree\" or \"table\"")
	}
	return nil
}

func (o *Options) run() (err error) {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	placementClient, err := clusterv1alpha1.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	placementList, err := placementClient.Placements(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	decisionList, err := placementClient.PlacementDecisions(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	if o.output == "table" {
		return o.printPlacementTable(placementList, decisionList)
	}
	return o.printPlacementTree(placementList, decisionList)
}

func (o *Options) printPlacementTree(placementList *v1alpha1.PlacementList, decisionList *v1alpha1.PlacementDecisionList) error {
	// save decisions into a map
	selectedClusters := make(map[string][]v1alpha1.ClusterDecision)
	for _, decision := range decisionList.Items {
		placementName := decision.ObjectMeta.Labels[placementLabel]
		selectedClusters[placementName] = decision.Status.Decisions
	}

	root := gotree.New("<Placement>")
	for _, pla := range placementList.Items {
		placementRoot := root.Add(color.New(color.FgBlue).Sprintf(pla.ObjectMeta.Name))

		statusNode := placementRoot.Add("<Status>")
		printPlacementStatusTree(statusNode, &pla.Status)

		// print selected clusters
		pnode := placementRoot.Add("<PlacementDecision>")
		clusters, ok := selectedClusters[pla.ObjectMeta.Name]
		if !ok {
			pnode.Add("<NoClusterSelected>")
		} else {
			for _, c := range clusters {
				pnode.Add(c.ClusterName)
			}
		}

	}

	fmt.Fprint(o.Streams.Out, root.Print())
	return nil
}

func printPlacementStatusTree(n gotree.Tree, status *v1alpha1.PlacementStatus) {
	con := n.Add("<Conditions>")
	testingConds := []string{
		v1alpha1.PlacementConditionSatisfied,
		v1alpha1.PlacementConditionMisconfigured,
	}
	sanitize := func(cond *metav1.Condition) string {
		if cond == nil {
			return "unknown"
		}
		if cond.Status == metav1.ConditionTrue {
			return "true"
		}
		return color.RedString("false")
	}

	for _, t := range testingConds {
		cond := meta.FindStatusCondition(status.Conditions, t)
		con.Add(fmt.Sprintf("%s -> %s", t, sanitize(cond)))
	}

	n.Add(fmt.Sprintf("<Number of Selected Clusters>: %v", status.NumberOfSelectedClusters))
}

func (o *Options) printPlacementTable(placementList *v1alpha1.PlacementList, decisionList *v1alpha1.PlacementDecisionList) error {
	o.printer = printers.NewTablePrinter(printers.PrintOptions{
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
	})

	// save decisions into a map
	selectedClusters := make(map[string][]v1alpha1.ClusterDecision)
	for _, decision := range decisionList.Items {
		placementName := decision.ObjectMeta.Labels[placementLabel]
		selectedClusters[placementName] = decision.Status.Decisions
	}

	//
	table, err := converToTable(placementList, selectedClusters)
	if err != nil {
		return err
	}

	return o.printer.PrintObj(table, o.Streams.Out)
}

func converToTable(placements *v1alpha1.PlacementList, selectedClusters map[string][]v1alpha1.ClusterDecision) (*metav1.Table, error) {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Status", Type: "string"},
			{Name: "Reason", Type: "string"},
			{Name: "SeletedClusters", Type: "array"},
		},
		Rows: []metav1.TableRow{},
	}

	for _, placement := range placements.Items {
		var clusters []string
		for _, i := range selectedClusters[placement.ObjectMeta.Name] {
			clusters = append(clusters, i.ClusterName)
		}

		row := convertRow(placement, clusters)
		table.Rows = append(table.Rows, row)
	}

	return table, nil
}

func convertRow(placement v1alpha1.Placement, clusters []string) metav1.TableRow {
	placementStatus := placement.Status.Conditions[0].Status
	statusReason := placement.Status.Conditions[0].Reason

	return metav1.TableRow{
		Cells:  []interface{}{placement.Name, placementStatus, statusReason, clusters},
		Object: runtime.RawExtension{Object: &placement},
	}
}
