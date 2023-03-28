// Copyright Contributors to the Open Cluster Management project
package addon

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	workapiv1 "open-cluster-management.io/api/work/v1"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonclient "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	workclient "open-cluster-management.io/api/client/work/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	o.printer.Competele()
	klog.V(1).InfoS("addon options:", "dry-run", o.ClusteradmFlags.DryRun, "clusters", o.ClusterOptions.AllClusters().List())
	o.addons = args

	return nil
}

func (o *Options) validate() error {
	if err := o.ClusteradmFlags.ValidateHub(); err != nil {
		return err
	}

	if err := o.printer.Validate(); err != nil {
		return err
	}

	if err := o.ClusterOptions.Validate(); err != nil {
		return err
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
	addonClient, err := addonclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	workClient, err := workclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	var clusters sets.String
	if o.ClusterOptions.AllClusters().Len() == 0 {
		clusters = sets.NewString()
		mcllist, err := clusterClient.ClusterV1().ManagedClusters().List(context.TODO(),
			metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, item := range mcllist.Items {
			clusters.Insert(item.ObjectMeta.Name)
		}
	} else {
		clusters = o.ClusterOptions.AllClusters()
	}

	cmaList, err := addonClient.AddonV1alpha1().ClusterManagementAddOns().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	addonList, err := addonClient.AddonV1alpha1().
		ManagedClusterAddOns(metav1.NamespaceAll).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	mcaByName := make(map[string][]addonv1alpha1.ManagedClusterAddOn)
	for _, addon := range addonList.Items {
		if _, ok := mcaByName[addon.Name]; !ok {
			mcaByName[addon.Name] = []addonv1alpha1.ManagedClusterAddOn{}
		}
		mcaByName[addon.Name] = append(mcaByName[addon.Name], addon)
	}

	workList, err := workClient.WorkV1().
		ManifestWorks(metav1.NamespaceAll).
		List(context.TODO(), metav1.ListOptions{
			LabelSelector: "open-cluster-management.io/addon-name",
		})
	if err != nil {
		return err
	}

	klog.V(3).InfoS("values:", "clusters", clusters)

	o.printer.WithTreeConverter(o.convertToTreeFunc(clusters.List(), mcaByName, workList.Items)).WithTableConverter(o.converToTableFunc(mcaByName))

	return o.printer.Print(o.Streams, cmaList)
}

func (o *Options) convertToTreeFunc(
	clusters []string,
	mcaByName map[string][]addonv1alpha1.ManagedClusterAddOn,
	works []workapiv1.ManifestWork,
) func(obj runtime.Object, tree *printer.TreePrinter) *printer.TreePrinter {
	return func(obj runtime.Object, tree *printer.TreePrinter) *printer.TreePrinter {
		if cmaList, ok := obj.(*addonv1alpha1.ClusterManagementAddOnList); ok {
			for _, cma := range cmaList.Items {
				if !shouldShow(o.addons, cma.Name) {
					continue
				}
				for _, addon := range mcaByName[cma.Name] {
					if !contains(clusters, addon.Namespace) {
						continue
					}
					conds := addonCondition(addon)
					tree.AddFileds(cma.Name, &conds)
					for _, work := range works {
						if contains(clusters, work.Namespace) && work.Labels["open-cluster-management.io/addon-name"] == addon.Name {
							workStatus := printer.WorkDetails(fmt.Sprintf(".%s.ManifestWork", addon.Namespace), &work)
							tree.AddFileds(cma.Name, &workStatus)
						}
					}
				}
			}
		}
		return tree
	}
}

func (o *Options) converToTableFunc(mcaByName map[string][]addonv1alpha1.ManagedClusterAddOn) func(obj runtime.Object) *metav1.Table {
	return func(obj runtime.Object) *metav1.Table {
		klog.V(3).InfoS("values:", "addons", mcaByName)
		table := &metav1.Table{
			ColumnDefinitions: []metav1.TableColumnDefinition{
				{Name: "Name", Type: "string"},
				{Name: "InstalledClusters", Type: "integer"},
			},
			Rows: []metav1.TableRow{},
		}

		if cmaList, ok := obj.(*addonv1alpha1.ClusterManagementAddOnList); ok {
			for _, cma := range cmaList.Items {
				if !shouldShow(o.addons, cma.Name) {
					continue
				}
				row := metav1.TableRow{
					Cells:  []interface{}{cma.Name, len(mcaByName[cma.Name])},
					Object: runtime.RawExtension{Object: &cma},
				}

				table.Rows = append(table.Rows, row)
			}
		}
		return table
	}
}

func addonCondition(addon addonv1alpha1.ManagedClusterAddOn) map[string]any {
	conds := map[string]any{}
	testingConds := []string{
		"Available",
		"ManifestApplied",
		"RegistrationApplied",
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
	for _, condType := range testingConds {
		cond := meta.FindStatusCondition(addon.Status.Conditions, condType)
		conds[fmt.Sprintf(".%s.Status.%s", addon.Namespace, condType)] = fmt.Sprintf("%s -> %s", condType, sanitize(cond))
	}
	return conds
}

func shouldShow(selectingAddons []string, addonName string) bool {
	if len(selectingAddons) == 0 { // empty list means all
		return true
	}
	return contains(selectingAddons, addonName)
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if target == v {
			return true
		}
	}
	return false
}
