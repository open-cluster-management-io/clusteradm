// Copyright Contributors to the Open Cluster Management project
package list

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	addonclient "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {

	klog.V(1).InfoS("list options:", "dry-run", o.ClusteradmFlags.DryRun, "clusters", o.clusters, "output-file", o.outputFile)

	return nil
}

func (o *Options) validate() (err error) {
	if o.clusters == "" {
		return fmt.Errorf("clusters is misisng")
	}
	return nil
}

func (o *Options) run() (err error) {
	alreadyProvidedClusters := make(map[string]bool)
	clusters := make([]string, 0)
	cs := strings.Split(o.clusters, ",")
	for _, c := range cs {
		if _, ok := alreadyProvidedClusters[c]; !ok {
			alreadyProvidedClusters[c] = true
			clusters = append(clusters, strings.TrimSpace(c))
		}
	}
	o.values.clusters = clusters

	klog.V(3).InfoS("values:", "clusters", o.values.clusters)

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

	return o.runWithClient(clusterClient, addonClient, kubeClient, apiExtensionsClient, dynamicClient, o.ClusteradmFlags.DryRun)
}

func (o *Options) runWithClient(clusterClient clusterclientset.Interface,
	addonClient addonclient.Interface,
	kubeClient kubernetes.Interface,
	apiExtensionsClient apiextensionsclient.Interface,
	dynamicClient dynamic.Interface,
	dryRun bool) error {

	for _, clusterName := range o.values.clusters {
		_, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(),
			clusterName,
			metav1.GetOptions{})
		if err != nil {
			return err
		}
	}

	output := make([]string, 0)

	for _, clusterName := range o.values.clusters {
		output = append(output, "CLUSTERNAME\tADDONNAME\tAVAILABLE\tDEGRADED\tPROGRESSING")

		list, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).List(context.TODO(),
			metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, item := range list.Items {
			temp := item.GetObjectMeta().GetNamespace() + "\t" + item.GetObjectMeta().GetName() + "\t"
			output = append(output, temp)
		}

		for _, out := range output {
			fmt.Println(out)
		}
	}
	return nil
}
