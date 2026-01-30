// Copyright Contributors to the Open Cluster Management project
package set

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterapiv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
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

	if len(o.Clusters) == 0 && len(o.Namespaces) == 0 {
		return fmt.Errorf("at least one cluster (--clusters) or namespace (--namespaces) must be specified")
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

	clusterSet, err := clusterClient.ClusterV1beta2().ManagedClusterSets().Get(context.TODO(), o.Clusterset, metav1.GetOptions{})
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

	if len(o.Namespaces) > 0 {
		if err := o.handleManagedNamespaces(clusterClient, clusterSet); err != nil {
			return err
		}
	}

	return nil
}

func (o *Options) handleManagedNamespaces(clusterClient *clusterclientset.Clientset, clusterSet *clusterv1beta2.ManagedClusterSet) error {
	fmt.Fprintf(o.Streams.Out, "Processing managed namespaces: %v\n", o.Namespaces)

	// Initialize ManagedNamespaces slice if nil
	if clusterSet.Spec.ManagedNamespaces == nil {
		clusterSet.Spec.ManagedNamespaces = []clusterapiv1.ManagedNamespaceConfig{}
	}

	// Build lookup map of existing namespace names
	existingMap := make(map[string]struct{}, len(clusterSet.Spec.ManagedNamespaces))
	for _, ns := range clusterSet.Spec.ManagedNamespaces {
		existingMap[ns.Name] = struct{}{}
	}

	var addedNamespaces []string
	var existingNamespaces []string

	for _, namespace := range o.Namespaces {
		namespace = strings.TrimSpace(namespace)
		// Validate namespace name (basic validation)
		if len(namespace) == 0 {
			fmt.Fprintf(o.Streams.ErrOut, "Warning: Empty namespace name provided, skipping\n")
			continue
		}

		// Check if namespace already exists
		if _, exists := existingMap[namespace]; exists {
			existingNamespaces = append(existingNamespaces, namespace)
			fmt.Fprintf(o.Streams.Out, "Managed namespace %s already exists in Clusterset %s\n", namespace, o.Clusterset)
			continue
		}

		// Add the new managed namespace
		clusterSet.Spec.ManagedNamespaces = append(clusterSet.Spec.ManagedNamespaces,
			clusterapiv1.ManagedNamespaceConfig{
				Name: namespace,
			})

		existingMap[namespace] = struct{}{}
		addedNamespaces = append(addedNamespaces, namespace)
	}
	if len(addedNamespaces) > 0 {
		// Update the clusterset with new managed namespaces
		updatedClusterSet, err := clusterClient.ClusterV1beta2().ManagedClusterSets().Update(
			context.TODO(),
			clusterSet,
			metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update clusterset with managed namespaces: %v", err)
		}

		// Print success messages
		fmt.Fprintf(o.Streams.Out, "\nSuccessfully updated Clusterset %s\n", o.Clusterset)
		fmt.Fprintf(o.Streams.Out, "Added %d managed namespace(s): %s\n",
			len(addedNamespaces), strings.Join(addedNamespaces, ", "))

		// Show total managed namespaces
		fmt.Fprintf(o.Streams.Out, "Total managed namespaces in clusterset: %d\n",
			len(updatedClusterSet.Spec.ManagedNamespaces))
	} else if len(existingNamespaces) > 0 {
		fmt.Fprintf(o.Streams.Out, "\nAll specified namespaces already exist in Clusterset %s\n", o.Clusterset)
	} else {
		fmt.Fprintf(o.Streams.Out, "\nNo changes made to Clusterset %s\n", o.Clusterset)
	}
	return nil
}
