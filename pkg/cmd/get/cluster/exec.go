// Copyright Contributors to the Open Cluster Management project
package cluster

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterapiv1 "open-cluster-management.io/api/cluster/v1"
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

	listOpt := metav1.ListOptions{}
	if len(o.Clusterset) != 0 {
		_, err := clusterClient.ClusterV1beta1().ManagedClusterSets().Get(context.TODO(), o.Clusterset, metav1.GetOptions{})
		if err != nil {
			return err
		}

		listOpt.LabelSelector = fmt.Sprintf("cluster.open-cluster-management.io/clusterset=%s", o.Clusterset)
	}

	clusters, err := clusterClient.ClusterV1().ManagedClusters().List(context.TODO(), listOpt)
	if err != nil {
		return err
	}

	table := converToTable(clusters)

	return o.printer.PrintObj(table, o.Streams.Out)
}

func converToTable(clusters *clusterapiv1.ManagedClusterList) *metav1.Table {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Accepted", Type: "boolean"},
			{Name: "Available", Type: "string"},
			{Name: "ClusterSet", Type: "string"},
			{Name: "CPU", Type: "string"},
			{Name: "Memory", Type: "string"},
			{Name: "Kubernetes Version", Type: "string"},
		},
		Rows: []metav1.TableRow{},
	}

	for _, cluster := range clusters.Items {
		row := convertRow(cluster)
		table.Rows = append(table.Rows, row)
	}

	return table
}

func convertRow(cluster clusterapiv1.ManagedCluster) metav1.TableRow {
	var available, cpu, memory, clusterset string

	availableCond := meta.FindStatusCondition(cluster.Status.Conditions, clusterapiv1.ManagedClusterConditionAvailable)
	if availableCond != nil {
		available = string(availableCond.Status)
	}

	if cpuResource, ok := cluster.Status.Capacity[clusterapiv1.ResourceCPU]; ok {
		cpu = cpuResource.String()
	}

	if memResource, ok := cluster.Status.Capacity[clusterapiv1.ResourceMemory]; ok {
		memory = memResource.String()
	}

	if len(cluster.Labels) > 0 {
		clusterset = cluster.Labels["cluster.open-cluster-management.io/clusterset"]
	}

	return metav1.TableRow{
		Cells:  []interface{}{cluster.Name, cluster.Spec.HubAcceptsClient, available, clusterset, cpu, memory, cluster.Status.Version.Kubernetes},
		Object: runtime.RawExtension{Object: &cluster},
	}
}
