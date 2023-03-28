// Copyright Contributors to the Open Cluster Management project
package enable

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stolostron/applier/pkg/apply"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/cmd/addon/enable/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

type ClusterAddonInfo struct {
	ClusterName string
	NameSpace   string
	AddonName   string
	Annotations map[string]string
}

func NewClusterAddonInfo(cn string, o *Options, an string) (ClusterAddonInfo, error) {
	// Parse provided annotations
	annos := map[string]string{}
	for _, annoString := range o.Annotate {
		annoSlice := strings.Split(annoString, "=")
		if len(annoSlice) != 2 {
			return ClusterAddonInfo{},
				fmt.Errorf("error parsing annotation '%s'. Expected to be of the form: key=value", annoString)
		}
		annos[annoSlice[0]] = annoSlice[1]
	}
	return ClusterAddonInfo{
		ClusterName: cn,
		NameSpace:   o.Namespace,
		AddonName:   an,
		Annotations: annos,
	}, nil
}

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("enable options:", "dry-run", o.ClusteradmFlags.DryRun, "names", o.Names, "clusters", o.ClusterOptions.AllClusters().List(), "output-file", o.OutputFile)

	return nil
}

func (o *Options) Validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	if len(o.Names) == 0 {
		return fmt.Errorf("names is missing")
	}

	if err := o.ClusterOptions.Validate(); err != nil {
		return err
	}

	return nil
}

func (o *Options) Run() error {
	addons := sets.NewString(o.Names...)
	clusters := o.ClusterOptions.AllClusters()

	klog.V(3).InfoS("values:", "addon", addons, "clusters", clusters)

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

	return o.runWithClient(clusterClient, kubeClient, apiExtensionsClient, dynamicClient, o.ClusteradmFlags.DryRun, addons.List(), clusters.List())
}

func (o *Options) runWithClient(clusterClient clusterclientset.Interface,
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

	output := make([]string, 0)
	reader := scenario.GetScenarioResourcesReader()

	applierBuilder := apply.NewApplierBuilder()
	applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()

	for _, addon := range addons {
		for _, clusterName := range clusters {
			cai, err := NewClusterAddonInfo(clusterName, o, addon)
			if err != nil {
				return err
			}
			out, err := applier.ApplyCustomResources(reader, cai, dryRun, "", "addons/addon.yaml")
			if err != nil {
				return err
			}
			output = append(output, out...)

			fmt.Fprintf(o.Streams.Out, "Deploying %s add-on to namespaces %s of managed cluster: %s.\n", addon, o.Namespace, clusterName)
		}
	}

	return apply.WriteOutput(o.OutputFile, output)
}
