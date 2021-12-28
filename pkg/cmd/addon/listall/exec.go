// Copyright Contributors to the Open Cluster Management project
package listall

import (
	"context"
	"fmt"

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

	klog.V(1).InfoS("listall options:", "dry-run", o.ClusteradmFlags.DryRun, "output-file", o.outputFile)

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

	return o.runWithClient(clusterClient, addonClient, kubeClient, apiExtensionsClient, dynamicClient, o.ClusteradmFlags.DryRun)
}

func (o *Options) runWithClient(clusterClient clusterclientset.Interface,
	addonClient addonclient.Interface,
	kubeClient kubernetes.Interface,
	apiExtensionsClient apiextensionsclient.Interface,
	dynamicClient dynamic.Interface,
	dryRun bool) error {

	mcllist, err := clusterClient.ClusterV1().ManagedClusters().List(context.TODO(),
		metav1.ListOptions{})
	if err != nil {
		return err
	}

	mcls := make([]string, 0)

	for _, item := range mcllist.Items {
		mcls = append(mcls, item.ObjectMeta.Name)
	}

	output := make([]string, 0)
	output = append(output, "CLUSTERNAME\tADDONNAME\tAVAILABLE\tDEGRADED\tPROGRESSING")

	for _, clusterName := range mcls {

		list, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).List(context.TODO(),
			metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, item := range list.Items {
			temp := item.GetObjectMeta().GetNamespace() + "\t" + item.GetObjectMeta().GetName() + "\t"
			output = append(output, temp)
		}
	}

	for _, out := range output {
		fmt.Println(out)
	}
	return nil
}
