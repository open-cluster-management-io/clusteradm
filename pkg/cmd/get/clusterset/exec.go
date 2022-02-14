// Copyright Contributors to the Open Cluster Management project
package clusterset

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterapiv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	return nil
}

func (o *Options) validate(args []string) (err error) {
	if len(args) != 0 {
		return fmt.Errorf("there should be no argument")
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

	clustersets, err := clusterClient.ClusterV1beta1().ManagedClusterSets().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	bindingMap := map[string][]string{}

	bindings, err := clusterClient.ClusterV1beta1().ManagedClusterSetBindings(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, binding := range bindings.Items {
		if _, ok := bindingMap[binding.Spec.ClusterSet]; !ok {
			bindingMap[binding.Spec.ClusterSet] = []string{}
		}

		bindingMap[binding.Spec.ClusterSet] = append(bindingMap[binding.Spec.ClusterSet], binding.Namespace)
	}

	table := converToTable(clustersets, bindingMap)

	return o.printer.PrintObj(table, o.Streams.Out)
}

func converToTable(clustersets *clusterapiv1beta1.ManagedClusterSetList, bindingMap map[string][]string) *metav1.Table {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Bound Namespaces", Type: "string"},
			{Name: "Status", Type: "string"},
		},
		Rows: []metav1.TableRow{},
	}

	for _, cluster := range clustersets.Items {
		bindings := bindingMap[cluster.Name]
		row := convertRow(cluster, bindings)
		table.Rows = append(table.Rows, row)
	}

	return table
}

func convertRow(clusterset clusterapiv1beta1.ManagedClusterSet, bindings []string) metav1.TableRow {
	var status string

	emptyCond := meta.FindStatusCondition(clusterset.Status.Conditions, clusterapiv1beta1.ManagedClusterSetConditionEmpty)
	if emptyCond != nil {
		status = string(emptyCond.Message)
	}

	return metav1.TableRow{
		Cells:  []interface{}{clusterset.Name, strings.Join(bindings, ","), status},
		Object: runtime.RawExtension{Object: &clusterset},
	}
}
