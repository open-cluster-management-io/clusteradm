// Copyright Contributors to the Open Cluster Management project
package set

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
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

func (o *Options) Validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	if len(o.Clusters) == 0 {
		return fmt.Errorf("cluster name must be specified in --clusters")
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

	_, err = clusterClient.ClusterV1beta2().ManagedClusterSets().Get(context.TODO(), o.Clusterset, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for _, clusterName := range o.Clusters {
		cluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), clusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if len(cluster.Labels) == 0 {
			cluster.Labels = map[string]string{}
		}

		clusterset := cluster.Labels["cluster.open-cluster-management.io/clusterset"]
		if clusterset == o.Clusterset {
			fmt.Fprintf(o.Streams.Out, "Cluster %s is already in Clusterset %s\n", clusterName, o.Clusterset)
			continue
		}

		cluster.Labels["cluster.open-cluster-management.io/clusterset"] = o.Clusterset
		_, err = clusterClient.ClusterV1().ManagedClusters().Update(context.TODO(), cluster, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		if len(clusterset) == 0 {
			fmt.Fprintf(o.Streams.Out, "Cluster %s is set to Clusterset %s\n", clusterName, o.Clusterset)
		} else {
			fmt.Fprintf(o.Streams.Out, "Cluster %s is set, from ClusterSet %s to Clusterset %s\n", clusterName, clusterset, o.Clusterset)
		}
	}

	return nil
}
