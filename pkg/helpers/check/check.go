// Copyright Contributors to the Open Cluster Management project
package check

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
)

const (
	ManagedClusterResourceName = "managedclusters"
	KlusterletResourceName     = "klusterlets"
	ClusterClaimResourceName   = "clusterclaims"
)

func CheckForHub(client clusterclient.Interface) error {
	msg := "hub oriented command should not running against non-hub cluster"

	list, err := client.Discovery().ServerResourcesForGroupVersion(clusterv1.GroupVersion.String())
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf(msg)

		}
		return fmt.Errorf("failed to list GroupVersion: %s", clusterv1.GroupVersion.String())

	}
	flag := findResource(list, ManagedClusterResourceName)
	if flag {
		return nil
	}
	return fmt.Errorf(msg)
}

func CheckForKlusterletCRD(client operatorclient.Interface) error {
	msg := "klusterlet crd not found"

	list, err := client.Discovery().ServerResourcesForGroupVersion(operatorv1.GroupVersion.String())
	if err != nil {
		return err
	}
	flag := findResource(list, KlusterletResourceName)
	if flag {
		return nil
	}
	return fmt.Errorf(msg)
}

func CheckForManagedCluster(client clusterclient.Interface) error {
	msg := "managed cluster oriented command should not running against non-managed cluster"

	list, err := client.Discovery().ServerResourcesForGroupVersion(clusterv1.GroupVersion.String())
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf(msg)

		}
		return fmt.Errorf("failed to list GroupVersion: %s", clusterv1.GroupVersion.String())

	}
	flag := findResource(list, ClusterClaimResourceName)
	if flag {
		return nil
	}
	return fmt.Errorf(msg)
}

func findResource(list *metav1.APIResourceList, resourceName string) bool {
	for _, item := range list.APIResources {
		if item.Name == resourceName {
			return true
		}
	}
	return false
}
