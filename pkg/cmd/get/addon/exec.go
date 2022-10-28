// Copyright Contributors to the Open Cluster Management project
package addon

import (
	"context"
	"fmt"

	"github.com/disiqueira/gotree"
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

	klog.V(1).InfoS("addon options:", "dry-run", o.ClusteradmFlags.DryRun, "clusters", o.clusters)
	o.addons = args

	return nil
}

func (o *Options) validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
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
	if len(o.clusters) == 0 {
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
		clusters = sets.NewString(o.clusters...)
	}

	klog.V(3).InfoS("values:", "clusters", clusters)

	return o.printAddonTree(clusters.List(), addonClient, workClient)
}

func (o *Options) printAddonTree(
	clusters []string,
	addonClient addonclient.Interface,
	workClient workclient.Interface) error {
	addonList, err := addonClient.AddonV1alpha1().
		ManagedClusterAddOns(metav1.NamespaceAll).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	addonByCluster := make(map[string][]*addonv1alpha1.ManagedClusterAddOn)
	for _, addon := range addonList.Items {
		if shouldShow(o.addons, &addon) {
			clusterName := addon.Namespace
			addon := addon
			addonByCluster[clusterName] = append(addonByCluster[clusterName], &addon)
		}
	}

	workList, err := workClient.WorkV1().
		ManifestWorks(metav1.NamespaceAll).
		List(context.TODO(), metav1.ListOptions{
			LabelSelector: "open-cluster-management.io/addon-name",
		})
	if err != nil {
		return err
	}

	root := gotree.New("<ManagedCluster>")
	for _, clusterName := range clusters {
		addonRoot := root.Add(color.New(color.FgBlue).Sprintf(clusterName))
		addons, ok := addonByCluster[clusterName]
		if !ok {
			continue
		}
		for _, addon := range addons {
			for _, work := range workList.Items {
				if clusterName == work.Namespace && work.Labels["open-cluster-management.io/addon-name"] == addon.Name {
					addonNode := addonRoot.Add(color.New(color.Bold).Sprintf("%s", addon.Name))
					statusNode := addonNode.Add("<Status>")
					printAddonStatus(statusNode, addon)
					workNode := addonNode.Add("<ManifestWork>")
					printer.PrintWorkDetail(workNode, &work)
				}
			}
		}
	}
	fmt.Fprint(o.Streams.Out, root.Print())
	return nil
}

func printAddonStatus(n gotree.Tree, addon *addonv1alpha1.ManagedClusterAddOn) {
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
		n.Add(fmt.Sprintf("%s -> %s", condType, sanitize(cond)))
	}
}

func shouldShow(selectingAddons []string, addon *addonv1alpha1.ManagedClusterAddOn) bool {
	if len(selectingAddons) == 0 { // empty list means all
		return true
	}
	return contains(selectingAddons, addon.Name)
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if target == v {
			return true
		}
	}
	return false
}
