// Copyright Contributors to the Open Cluster Management project
package addon

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"open-cluster-management.io/api/addon/v1alpha1"
	addonclient "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {

	klog.V(1).InfoS("addon options:", "dry-run", o.ClusteradmFlags.DryRun, "clusters", o.clusters)

	return nil
}

func (o *Options) validate() (err error) {
	return nil
}

func (o *Options) run() (err error) {
	clusters := make([]string, 0)

	if len(o.clusters) == 0 {
		klog.V(3).InfoS("values:", "all clusters")
	} else {
		alreadyProvidedClusters := make(map[string]bool)
		cs := strings.Split(o.clusters, ",")
		for _, c := range cs {
			if _, ok := alreadyProvidedClusters[c]; !ok {
				alreadyProvidedClusters[c] = true
				clusters = append(clusters, strings.TrimSpace(c))
			}
		}

		klog.V(3).InfoS("values:", "clusters", clusters)
	}

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

	return o.runWithClient(clusterClient, addonClient, kubeClient, apiExtensionsClient, dynamicClient, o.ClusteradmFlags.DryRun, clusters)
}

func (o *Options) runWithClient(clusterClient clusterclientset.Interface,
	addonClient addonclient.Interface,
	kubeClient kubernetes.Interface,
	apiExtensionsClient apiextensionsclient.Interface,
	dynamicClient dynamic.Interface,
	dryRun bool,
	clusters []string) error {

	if len(clusters) == 0 {
		mcllist, err := clusterClient.ClusterV1().ManagedClusters().List(context.TODO(),
			metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, item := range mcllist.Items {
			clusters = append(clusters, item.ObjectMeta.Name)
		}
	} else {
		for _, clusterName := range clusters {
			_, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(),
				clusterName,
				metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
	}

	var addons []v1alpha1.ManagedClusterAddOn

	for _, clusterName := range clusters {

		list, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).List(context.TODO(),
			metav1.ListOptions{})
		if err != nil {
			return err
		}

		addons = append(addons, list.Items...)
	}

	fmt.Fprintf(o.Streams.Out, "ADDONNAME\tCLUSTERNAME\n")

	// sort by addon name, then by cluster name
	sort.Slice(addons, func(i, j int) bool {
		if strings.Compare(addons[i].GetObjectMeta().GetName(), addons[j].GetObjectMeta().GetName()) == -1 {
			return true
		}
		if strings.Compare(addons[i].GetObjectMeta().GetName(), addons[j].GetObjectMeta().GetName()) == 0 &&
			strings.Compare(addons[i].GetObjectMeta().GetNamespace(), addons[j].GetObjectMeta().GetNamespace()) == -1 {
			return true
		}
		return false
	})

	for _, item := range addons {
		fmt.Fprintf(o.Streams.Out, "%s\t%s\n", item.GetObjectMeta().GetName(), item.GetObjectMeta().GetNamespace())
	}

	return nil
}
