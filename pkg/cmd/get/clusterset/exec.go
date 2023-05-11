// Copyright Contributors to the Open Cluster Management project
package clusterset

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	v1 "open-cluster-management.io/api/cluster/v1"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterapiv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	clusterClient, err := clusterclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	o.Client = clusterClient

	o.printer.Competele()

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

func (o *Options) run() (err error) {
	clustersets, err := o.Client.ClusterV1beta2().ManagedClusterSets().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	o.printer.WithTreeConverter(o.convertToTree).WithTableConverter(o.converToTable)

	return o.printer.Print(o.Streams, clustersets)
}

func (o *Options) convertToTree(obj runtime.Object, tree *printer.TreePrinter) *printer.TreePrinter {
	bindingMap := map[string][]string{}
	bindings, err := o.Client.ClusterV1beta2().ManagedClusterSetBindings(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, binding := range bindings.Items {
		if _, ok := bindingMap[binding.Spec.ClusterSet]; !ok {
			bindingMap[binding.Spec.ClusterSet] = []string{}
		}

		bindingMap[binding.Spec.ClusterSet] = append(bindingMap[binding.Spec.ClusterSet], binding.Namespace)
	}

	getter := &clusterGetter{client: o.Client}

	if csList, ok := obj.(*clusterapiv1beta2.ManagedClusterSetList); ok {
		for _, clusterset := range csList.Items {
			boundNs, status := getFileds(clusterset, bindingMap[clusterset.Name])
			clusters, err := getter.listClustersByClusterSet(&clusterset)
			if err != nil {
				klog.Fatalf("Failed to list cluster in clusterset %s: %v", clusterset.Name, err)
			}
			mp := make(map[string]interface{})
			mp[".BoundNamespace"] = boundNs
			mp[".Status"] = status
			mp[".Clusters"] = clusters
			tree.AddFileds(clusterset.Name, &mp)
		}
	}

	return tree
}

func (o *Options) converToTable(obj runtime.Object) *metav1.Table {
	bindingMap := map[string][]string{}
	bindings, err := o.Client.ClusterV1beta2().ManagedClusterSetBindings(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, binding := range bindings.Items {
		if _, ok := bindingMap[binding.Spec.ClusterSet]; !ok {
			bindingMap[binding.Spec.ClusterSet] = []string{}
		}

		bindingMap[binding.Spec.ClusterSet] = append(bindingMap[binding.Spec.ClusterSet], binding.Namespace)
	}

	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Bound Namespaces", Type: "string"},
			{Name: "Status", Type: "string"},
		},
		Rows: []metav1.TableRow{},
	}

	if csList, ok := obj.(*clusterapiv1beta2.ManagedClusterSetList); ok {
		for _, clusterset := range csList.Items {
			boundNs, status := getFileds(clusterset, bindingMap[clusterset.Name])
			row := metav1.TableRow{
				Cells:  []interface{}{clusterset.Name, boundNs, status},
				Object: runtime.RawExtension{Object: &clusterset},
			}

			table.Rows = append(table.Rows, row)
		}
	}

	return table
}

func getFileds(clusterset clusterapiv1beta2.ManagedClusterSet, bindings []string) (boundNs, status string) {
	boundNs = strings.Join(bindings, ",")

	emptyCond := meta.FindStatusCondition(clusterset.Status.Conditions, clusterapiv1beta2.ManagedClusterSetConditionEmpty)
	if emptyCond != nil {
		status = emptyCond.Message
	}

	return
}

type clusterGetter struct {
	client clusterclientset.Interface
}

func (c *clusterGetter) List(selector labels.Selector) ([]*v1.ManagedCluster, error) {
	var ret []*v1.ManagedCluster
	clusters, err := c.client.ClusterV1().ManagedClusters().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	for _, cluster := range clusters.Items {
		ret = append(ret, cluster.DeepCopy())
	}
	return ret, nil
}

func (c *clusterGetter) listClustersByClusterSet(clusterset *clusterapiv1beta2.ManagedClusterSet) ([]string, error) {
	clusters, err := clusterapiv1beta2.GetClustersFromClusterSet(clusterset, c)
	if err != nil {
		return nil, err
	}

	var ret []string
	for _, cluster := range clusters {
		ret = append(ret, cluster.Name)
	}
	return ret, nil
}
