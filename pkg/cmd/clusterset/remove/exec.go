// Copyright Contributors to the Open Cluster Management project
package remove

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
)

const (
	clusterSetLabel = "cluster.open-cluster-management.io/clusterset"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	if len(args) == 0 {
		return fmt.Errorf("the name of the clusterset must be specified")
	}

	if len(args) > 1 {
		return fmt.Errorf("only one clusterset can be specified")
	}

	o.Clusterset = args[0]

	return nil
}

func (o *Options) validate() (err error) {
	if len(o.Clusters) == 0 {
		return fmt.Errorf("cluster name must be specified in --clusters")
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

	for _, clusterName := range o.Clusters {
		cluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), clusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if clusterset := cluster.Labels[clusterSetLabel]; clusterset != o.Clusterset {
			fmt.Fprintf(o.Streams.Out, "Cluster %s is not in Clusterset %s\n", clusterName, o.Clusterset)
			continue
		}

		// if cluster is really in the specified clusterset, delete the label.
		delete(cluster.Labels, clusterSetLabel)
		_, err = clusterClient.ClusterV1().ManagedClusters().Update(context.TODO(), cluster, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		fmt.Fprintf(o.Streams.Out, "Cluster %s is removed from Clusterset %s\n", clusterName, o.Clusterset)
	}

	return nil
}
