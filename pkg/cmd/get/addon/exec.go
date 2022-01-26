// Copyright Contributors to the Open Cluster Management project
package addon

import (
	"context"
	"sort"

	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"open-cluster-management.io/api/addon/v1alpha1"
	addonclient "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers"

	"k8s.io/cli-runtime/pkg/printers"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {

	klog.V(1).InfoS("addon options:", "dry-run", o.ClusteradmFlags.DryRun, "clusters", o.clusters)

	return nil
}

func (o *Options) validate() (err error) {
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

	kubeClient, apiExtensionsClient, dynamicClient, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
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

	return o.runWithClient(clusterClient, addonClient, kubeClient, apiExtensionsClient, dynamicClient, o.ClusteradmFlags.DryRun, clusters.List())
}

func (o *Options) runWithClient(clusterClient clusterclientset.Interface,
	addonClient addonclient.Interface,
	kubeClient kubernetes.Interface,
	apiExtensionsClient apiextensionsclient.Interface,
	dynamicClient dynamic.Interface,
	dryRun bool,
	clusters []string) error {

	for _, clusterName := range clusters {
		_, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(),
			clusterName,
			metav1.GetOptions{})
		if err != nil {
			return err
		}
	}

	var addonlist v1alpha1.ManagedClusterAddOnList
	for _, clusterName := range clusters {

		list, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).List(context.TODO(),
			metav1.ListOptions{})
		if err != nil {
			return err
		}

		addonlist.Items = append(addonlist.Items, list.Items...)
	}

	printer := printers.NewTablePrinter(printers.PrintOptions{
		NoHeaders:     false,
		WithNamespace: true,
		WithKind:      false,
		Wide:          false,
		ShowLabels:    false,
		Kind: schema.GroupKind{
			Group: "addon.open-cluster-management.io",
			Kind:  "ManagedClusterAddOn",
		},
		ColumnLabels:     []string{},
		SortBy:           "",
		AllowMissingKeys: true,
	})

	sort.Slice(addonlist.Items, func(i, j int) bool {
		return addonlist.Items[i].Name < addonlist.Items[j].Name
	})

	printer.PrintObj(&addonlist, o.Streams.Out)
	return nil
}
