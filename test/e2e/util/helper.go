// Copyright Contributors to the Open Cluster Management project
package util

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
)

// WaitNamespaceDeleted receive a kubeconfigpath, a context name and a namespace name,
// then poll until the specific namespace is fully deleted or an error occurs.
func WaitNamespaceDeleted(restcfg *rest.Config, namespace string) error {
	clientset, err := kubernetes.NewForConfig(restcfg)
	if err != nil {
		return err
	}

	return wait.PollUntilContextCancel(context.TODO(), 1*time.Second, true, func(ctx context.Context) (bool, error) {
		ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		fmt.Printf("namespace stil exists %v\n", ns.Status)
		return false, nil
	})
}

func WaitForManagedClusterAvailableStatusToChange(restcfg *rest.Config, clusterName string) error {
	klusterClient, err := clusterclient.NewForConfig(restcfg)
	if err != nil {
		return err
	}
	return wait.PollUntilContextCancel(context.TODO(), 1*time.Second, true, func(ctx context.Context) (bool, error) {
		managedCluster, err := klusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), clusterName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		for _, cond := range managedCluster.Status.Conditions {
			if cond.Type == "ManagedClusterConditionAvailable" && cond.Status != "True" {
				return true, nil
			}
		}

		return false, nil
	})
}

func DeleteClusterCSRs(restcfg *rest.Config) error {
	clientset, err := kubernetes.NewForConfig(restcfg)
	if err != nil {
		return err
	}

	return clientset.CertificatesV1().CertificateSigningRequests().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: "open-cluster-management.io/cluster-name",
	})
}

func DeleteClusterFinalizers(restcfg *rest.Config) error {
	clientset, err := clusterclient.NewForConfig(restcfg)
	if err != nil {
		return err
	}

	clusterList, err := clientset.ClusterV1().ManagedClusters().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, mcl := range clusterList.Items {
		mcl.Finalizers = []string{}
		_, err := clientset.ClusterV1().ManagedClusters().Update(context.TODO(), &mcl, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

// buildConfigFromFlags build rest config for specified context in the kubeconfigfile.
func buildConfigFromFlags(context, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}
