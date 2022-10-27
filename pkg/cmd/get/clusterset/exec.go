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
	clustersets, err := o.Client.ClusterV1beta1().ManagedClusterSets().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	o.printer.WithTreeConverter(o.convertToTree).WithTableConverter(o.converToTable)

	return o.printer.Print(o.Streams, clustersets)
}

func (o *Options) convertToTree(obj runtime.Object, tree *printer.TreePrinter) *printer.TreePrinter {
	bindingMap := map[string][]string{}
	bindings, err := o.Client.ClusterV1beta1().ManagedClusterSetBindings(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, binding := range bindings.Items {
		if _, ok := bindingMap[binding.Spec.ClusterSet]; !ok {
			bindingMap[binding.Spec.ClusterSet] = []string{}
		}

		bindingMap[binding.Spec.ClusterSet] = append(bindingMap[binding.Spec.ClusterSet], binding.Namespace)
	}

	if csList, ok := obj.(*clusterapiv1beta1.ManagedClusterSetList); ok {
		for _, clusterset := range csList.Items {
			boundNs, status := getFileds(clusterset, bindingMap[clusterset.Name])
			mp := make(map[string]interface{})
			mp[".BoundNamespace"] = boundNs
			mp[".Status"] = status
			tree.AddFileds(clusterset.Name, &mp)
		}
	}

	return tree
}

func (o *Options) converToTable(obj runtime.Object) *metav1.Table {
	bindingMap := map[string][]string{}
	bindings, err := o.Client.ClusterV1beta1().ManagedClusterSetBindings(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
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

	if csList, ok := obj.(*clusterapiv1beta1.ManagedClusterSetList); ok {
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

func getFileds(clusterset clusterapiv1beta1.ManagedClusterSet, bindings []string) (boundNs, status string) {
	boundNs = strings.Join(bindings, ",")

	emptyCond := meta.FindStatusCondition(clusterset.Status.Conditions, clusterapiv1beta1.ManagedClusterSetConditionEmpty)
	if emptyCond != nil {
		status = string(emptyCond.Message)
	}

	return
}
