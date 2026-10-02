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
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
)

func (o *Options) complete(_ *cobra.Command, _ []string) (err error) {
	o.printer.Complete()

	return nil
}

func (o *Options) validate(args []string) (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	if len(args) != 0 {
		return fmt.Errorf("there should be no argument")
	}

	err = o.printer.Validate()
	if err != nil {
		return err
	}

	return nil
}

func (o *Options) run(ctx context.Context) (err error) {
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
		_, err := clusterClient.ClusterV1beta2().ManagedClusterSets().Get(ctx, o.Clusterset, metav1.GetOptions{})
		if err != nil {
			return err
		}

		listOpt.LabelSelector = fmt.Sprintf("cluster.open-cluster-management.io/clusterset=%s", o.Clusterset)
	}

	clusters, err := clusterClient.ClusterV1().ManagedClusters().List(ctx, listOpt)
	if err != nil {
		return err
	}

	o.printer.WithTreeConverter(o.convertToTree).WithTableConverter(o.convertToTable)

	return o.printer.Print(o.Streams, clusters)
}

func (o *Options) convertToTree(obj runtime.Object, tree *printer.TreePrinter) (*printer.TreePrinter, error) {
	if mclList, ok := obj.(*clusterapiv1.ManagedClusterList); ok {
		for _, cluster := range mclList.Items {
			accepted, available, version, cpu, memory, clusterset := getFields(cluster)
			mp := make(map[string]interface{})
			mp[".Accepted"] = accepted
			mp[".Available"] = available
			mp[".ClusterSet"] = clusterset
			mp[".KubernetesVersion"] = version
			mp[".Capacity.Cpu"] = cpu
			mp[".Capacity.Memory"] = memory

			tree.AddFileds(cluster.Name, &mp)
		}
	}
	return tree, nil
}

func (o *Options) convertToTable(obj runtime.Object) (*metav1.Table, error) {
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

	if mclList, ok := obj.(*clusterapiv1.ManagedClusterList); ok {
		for _, cluster := range mclList.Items {
			accepted, available, version, cpu, memory, clusterset := getFields(cluster)
			row := metav1.TableRow{
				Cells:  []interface{}{cluster.Name, accepted, available, clusterset, cpu, memory, version},
				Object: runtime.RawExtension{Object: &cluster},
			}

			table.Rows = append(table.Rows, row)
		}
	}
	return table, nil
}

func getFields(cluster clusterapiv1.ManagedCluster) (accepted bool, available, version, cpu, memory, clusterset string) {
	accepted = cluster.Spec.HubAcceptsClient

	version = cluster.Status.Version.Kubernetes

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

	return
}
