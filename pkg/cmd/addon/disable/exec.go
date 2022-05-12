// Copyright Contributors to the Open Cluster Management project
package disable

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	addonclient "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("disable options:", "dry-run", o.ClusteradmFlags.DryRun, "names", o.Names, "clusters", o.Clusters, "all-clusters", o.Allclusters)

	return nil
}

func (o *Options) Validate() (err error) {
	if len(o.Names) == 0 {
		return fmt.Errorf("names is missing")
	}

	if !o.Allclusters && len(o.Clusters) == 0 {
		return fmt.Errorf("clusters is missing")
	}

	if o.Allclusters && len(o.Clusters) != 0 {
		return fmt.Errorf("flag --all-clusters and --clusters can not be set together")
	}
	return nil
}

func (o *Options) Run() (err error) {
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

	addons := sets.NewString(o.Names...)

	var clusters sets.String
	if o.Allclusters {
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
		clusters = sets.NewString(o.Clusters...)
	}

	klog.V(3).InfoS("addon to be disabled with cluster values:", "addon", addons.List(), "clusters", clusters.List())

	return o.runWithClient(clusterClient, addonClient, kubeClient, apiExtensionsClient, dynamicClient, o.ClusteradmFlags.DryRun, addons.List(), clusters.List())
}

func (o *Options) runWithClient(clusterClient clusterclientset.Interface,
	addonClient addonclient.Interface,
	kubeClient kubernetes.Interface,
	apiExtensionsClient apiextensionsclient.Interface,
	dynamicClient dynamic.Interface,
	dryRun bool,
	addons []string,
	clusters []string) error {

	for _, clusterName := range clusters {
		_, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(),
			clusterName,
			metav1.GetOptions{})
		if err != nil {
			return err
		}
	}

	for _, addon := range addons {
		for _, clusterName := range clusters {
			err := addonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).Delete(context.TODO(),
				addon,
				metav1.DeleteOptions{})
			if err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
			}
			fmt.Fprintf(o.Streams.Out, "Undeploying %s add-on in managed cluster: %s.\n", addon, clusterName)
		}
	}

	return nil
}
