// Copyright Contributors to the Open Cluster Management project
package enable

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/cmd/addon/enable/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"
)

const appMgrAddonName = "application-manager"

//ClusterName: The cluster name used in the template
type ClusterName struct {
	ClusterName string
	NameSpace   string
}

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("addon options:", "dry-run", o.ClusteradmFlags.DryRun, "names", o.names, "clusters", o.clusters, "output-file", o.outputFile)

	return nil
}

func (o *Options) validate() error {
	if o.names == "" {
		return fmt.Errorf("names is missing")
	}

	names := strings.Split(o.names, ",")
	for _, n := range names {
		if n != appMgrAddonName {
			return fmt.Errorf("invalid add-on name %s", n)
		}
	}

	if o.clusters == "" {
		return fmt.Errorf("clusters is misisng")
	}

	return nil
}

func (o *Options) run() error {
	alreadyProvidedAddons := make(map[string]bool)
	addons := make([]string, 0)
	names := strings.Split(o.names, ",")
	for _, n := range names {
		if _, ok := alreadyProvidedAddons[n]; !ok {
			alreadyProvidedAddons[n] = true
			addons = append(addons, strings.TrimSpace(n))
		}
	}
	o.values.addons = addons

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

	klog.V(3).InfoS("values:", "addon", o.values.addons, "clusters", o.values.clusters)

	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	clusterClient, err := clusterclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	kubeClient, apiExtensionsClient, dynamicClient, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	return o.runWithClient(clusterClient, kubeClient, apiExtensionsClient, dynamicClient, o.ClusteradmFlags.DryRun)
}

func (o *Options) runWithClient(clusterClient clusterclientset.Interface,
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
	reader := scenario.GetScenarioResourcesReader()

	applierBuilder := &apply.ApplierBuilder{}
	applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()

	for _, addon := range o.values.addons {
		if addon == appMgrAddonName {
			for _, clusterName := range o.values.clusters {
				cn := &ClusterName{ClusterName: clusterName, NameSpace: o.namespace}

				out, err := applier.ApplyCustomResources(reader, cn, dryRun, "", "addons/appmgr/addon.yaml")
				if err != nil {
					return err
				}
				output = append(output, out...)

				fmt.Printf("Deploying %s add-on to namespaces %s of managed cluster: %s.\n", appMgrAddonName, o.namespace, clusterName)
			}
		}
	}

	return apply.WriteOutput(o.outputFile, output)
}
