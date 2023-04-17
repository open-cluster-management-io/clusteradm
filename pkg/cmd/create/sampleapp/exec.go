// Copyright Contributors to the Open Cluster Management project

package sampleapp

import (
	"context"
	"errors"
	"fmt"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/cmd/create/sampleapp/scenario"
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

	f := o.ClusteradmFlags.KubectlFactory
	o.builder = f.NewBuilder()

	return nil
}

func (o *Options) Validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
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

	return o.runWithClient(clusterClient, o.ClusteradmFlags.DryRun)
}

func (o *Options) runWithClient(clusterClient clusterclientset.Interface, dryRun bool) error {

	// Label all managed clusters with clusterset and placement labels
	err := o.checkManagedClusterBinding(clusterClient, dryRun)
	if err != nil {
		return err
	}

	// Apply sample application manifest to hub cluster
	err = o.deployApp()
	if err != nil {
		return err
	}

	return nil
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

func (o *Options) deployApp() error {
	// Prepare deployment tools
	r := reader.NewResourceReader(o.builder, o.ClusteradmFlags.DryRun, o.Streams)

	// Retrieve sample application manifests
	_, currentFilePath, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("Error retrieving current file path")
	}
	appDir := filepath.Join(filepath.Dir(currentFilePath), pathToAppManifests)
	files, err := filePathWalkDir(appDir)
	if err != nil {
		return err
	}

	// Apply manifests
	return r.Apply(scenario.Files, o, files...)
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
