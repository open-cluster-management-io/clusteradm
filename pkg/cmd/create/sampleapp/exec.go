// Copyright Contributors to the Open Cluster Management project

package sampleapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/cmd/create/sampleapp/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"
)

const (
	defaultSampleAppName = "sampleapp"
	pathToAppManifests   = "scenario/sampleapp"
	clusterSetLabel      = "cluster.open-cluster-management.io/clusterset"
	placementLabel       = "placement"
	placementLabelValue  = "sampleapp"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {

	if len(args) > 1 {
		return fmt.Errorf("only one sample app name can be specified")
	}

	if len(args) == 0 {
		o.SampleAppName = defaultSampleAppName
	} else {
		o.SampleAppName = args[0]
	}

	return nil
}

func (o *Options) Validate() (err error) {

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

	// Label all managed clusters with clusterset and placement labels
	err := o.checkManagedClusterBinding(clusterClient, dryRun)
	if err != nil {
		return err
	}

	// Apply sample application manifest to hub cluster
	output, err := o.deployApp(kubeClient, apiExtensionsClient, dynamicClient, dryRun)
	if err != nil {
		return err
	}

	// Print generated manifest to console if runtime is flaged as DryRun
	if dryRun {
		var dryRunOutput string
		for _, s := range output {
			dryRunOutput += fmt.Sprintf("%s\n---\n", s)
		}
		fmt.Print(dryRunOutput)
	}

	return apply.WriteOutput(o.OutputFile, output)
}

func (o *Options) checkManagedClusterBinding(clusterClient clusterclientset.Interface, dryRun bool) error {

	// Skip if dryRun
	if dryRun {
		return nil
	}

	// Get managed clusters
	clusters, err := clusterClient.ClusterV1().ManagedClusters().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	// Check for binding labels in managed clusters
	for _, cluster := range clusters.Items {
		managedCluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), cluster.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if cs, ok := managedCluster.Labels[clusterSetLabel]; !ok || (cs != fmt.Sprintf("app-%s", o.SampleAppName)) {
			fmt.Fprintf(o.Streams.Out, "[WARNING] Label \"%s=%s\" has not been found in ManagedCluster %s, could not establish binding.\n", clusterSetLabel, fmt.Sprintf("app-%s", o.SampleAppName), cluster.Name)
		}
		if p, ok := managedCluster.Labels[placementLabel]; !ok || (p != placementLabelValue) {
			fmt.Fprintf(o.Streams.Out, "[WARNING] Label \"%s=%s\" has not been found in ManagedCluster %s, could not establish binding.\n", placementLabel, placementLabelValue, cluster.Name)
		}
	}

	return nil
}

func (o *Options) deployApp(kubeClient kubernetes.Interface,
	apiExtensionsClient apiextensionsclient.Interface,
	dynamicClient dynamic.Interface,
	dryRun bool) ([]string, error) {

	// Prepare deployment tools
	reader := scenario.GetScenarioResourcesReader()
	applierBuilder := apply.NewApplierBuilder()
	applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()

	// Retrieve sample application manifests
	_, currentFilePath, _, ok := runtime.Caller(0)
	if !ok {
		return nil, errors.New("Error retrieving current file path")
	}
	appDir := filepath.Join(filepath.Dir(currentFilePath), pathToAppManifests)
	files, err := filePathWalkDir(appDir)
	if err != nil {
		return nil, err
	}

	// Apply manifests
	return applier.ApplyCustomResources(reader, o, dryRun, "", files...)
}

func filePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			relPath, err := filepath.Rel(filepath.Dir(root), path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		return nil
	})
	return files, err
}
