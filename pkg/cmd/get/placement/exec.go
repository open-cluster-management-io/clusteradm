// Copyright Contributors to the Open Cluster Management project
package placement

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clusterv1beta1 "open-cluster-management.io/api/client/cluster/clientset/versioned/typed/cluster/v1beta1"
	"open-cluster-management.io/api/cluster/v1beta1"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
)

const placementLabel = "cluster.open-cluster-management.io/placement"

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	o.printer.Competele()

	return nil
}

func (o *Options) validate(args []string) (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	if len(args) > 1 {
		return fmt.Errorf("the number of placement name should be 0 or 1")
	}
	if len(args) == 1 {
		o.PlacementName = args[0]
	}

	err = o.printer.Validate()
	if err != nil {
		return err
	}

	return nil
}

func (o *Options) run() (err error) {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}

	if o.Namespace != "" {
		nsClient, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return err
		}
		_, err = nsClient.CoreV1().Namespaces().Get(context.TODO(), o.Namespace, metav1.GetOptions{})
		if err != nil {
			return err
		}
	} else {
		o.Namespace = metav1.NamespaceAll
	}

	placementClient, err := clusterv1beta1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	o.Client = placementClient

	var placementList *v1beta1.PlacementList
	if o.PlacementName == "" {
		placementList, err = o.Client.Placements(o.Namespace).List(context.TODO(), metav1.ListOptions{})
	} else {
		placementList, err = o.Client.Placements(o.Namespace).List(context.TODO(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", o.PlacementName),
		})
	}
	if err != nil {
		return err
	}

	o.printer.WithTreeConverter(o.convertToTree).WithTableConverter(o.converToTable)

	return o.printer.Print(o.Streams, placementList)
}

func (o *Options) convertToTree(obj runtime.Object, tree *printer.TreePrinter) *printer.TreePrinter {
	decisionList, err := o.Client.PlacementDecisions(o.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	// save decisions into a map
	selectedClusters := make(map[string][]v1beta1.ClusterDecision)
	for _, decision := range decisionList.Items {
		placementName := decision.ObjectMeta.Labels[placementLabel]
		selectedClusters[placementName] = decision.Status.Decisions
	}

	if placementList, ok := obj.(*v1beta1.PlacementList); ok {
		for _, pla := range placementList.Items {
			mp := make(map[string]interface{})
			namespace, clusterset, satisfied, misconfig, number, decision := getFileds(pla, selectedClusters)
			mp[".Namespace"] = namespace
			mp[".ClusterSet"] = clusterset
			mp[".Status.NumberOfSelectedClusters"] = number
			mp[".Status.Conditions.PlacementConditionSatisfied"] = satisfied
			mp[".Status.Conditions.PlacementConditionMisconfigured"] = misconfig
			mp[".PlacementDecision"] = decision

			tree.AddFileds(pla.ObjectMeta.Name, &mp)
		}
	}

	return tree
}

func getFileds(placement v1beta1.Placement, selectedClusters map[string][]v1beta1.ClusterDecision) (namespace string, clusterset []string, satisfied string, misconfig string, number int, decision []string) {
	namespace = placement.Namespace
	clusterset = placement.Spec.ClusterSets

	sanitize := func(cond *metav1.Condition) string {
		if cond == nil {
			return "unknown"
		}
		if cond.Status == metav1.ConditionTrue {
			return "true"
		}
		return color.RedString("false")
	}

	cond := meta.FindStatusCondition(placement.Status.Conditions, v1beta1.PlacementConditionSatisfied)
	satisfied = sanitize(cond)
	cond = meta.FindStatusCondition(placement.Status.Conditions, v1beta1.PlacementConditionMisconfigured)
	misconfig = sanitize(cond)

	number = int(placement.Status.NumberOfSelectedClusters)

	clusters, ok := selectedClusters[placement.ObjectMeta.Name]
	if !ok {
		decision = []string{"NoClusterSelected"}
	} else {
		for _, i := range clusters {
			decision = append(decision, i.ClusterName)
		}
	}

	return
}

func (o *Options) converToTable(obj runtime.Object) *metav1.Table {
	decisionList, err := o.Client.PlacementDecisions(o.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	// save decisions into a map
	selectedClusters := make(map[string][]v1beta1.ClusterDecision)
	for _, decision := range decisionList.Items {
		placementName := decision.ObjectMeta.Labels[placementLabel]
		selectedClusters[placementName] = decision.Status.Decisions
	}

	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Status", Type: "string"},
			{Name: "Reason", Type: "string"},
			{Name: "SeletedClusters", Type: "array"},
		},
		Rows: []metav1.TableRow{},
	}

	if placementList, ok := obj.(*v1beta1.PlacementList); ok {
		for _, placement := range placementList.Items {
			var clusters []string
			for _, i := range selectedClusters[placement.ObjectMeta.Name] {
				clusters = append(clusters, i.ClusterName)
			}

			row := convertRow(placement, clusters)
			table.Rows = append(table.Rows, row)
		}
	}

	return table
}

func convertRow(placement v1beta1.Placement, clusters []string) metav1.TableRow {
	placementStatus := placement.Status.Conditions[0].Status
	statusReason := placement.Status.Conditions[0].Reason

	return metav1.TableRow{
		Cells:  []interface{}{placement.Name, placementStatus, statusReason, clusters},
		Object: runtime.RawExtension{Object: &placement},
	}
}
