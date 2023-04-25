// Copyright Contributors to the Open Cluster Management project
package util

import (
	"context"
	"fmt"
	"time"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
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
func WaitNamespaceDeleted(kubeconfigpath string, ctx string, namespace string) error {
	restcfg, err := buildConfigFromFlags(ctx, kubeconfigpath)
	if err != nil {
		return fmt.Errorf("error occurred while build rest config: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(restcfg)
	if err != nil {
		return err
	}

	return wait.PollUntilContextCancel(context.TODO(), 1*time.Second, true, func(ctx context.Context) (bool, error) {
		_, err = clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	})
}

func DeleteClusterCSRs(kubeconfigpath string, ctx string) error {
	restcfg, err := buildConfigFromFlags(ctx, kubeconfigpath)
	if err != nil {
		return fmt.Errorf("error occurred while build rest config: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(restcfg)
	if err != nil {
		return err
	}

	return clientset.CertificatesV1().CertificateSigningRequests().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: "open-cluster-management.io/cluster-name",
	})
}

func DeleteClusterFinalizers(kubeconfigpath string, ctx string) error {
	restcfg, err := buildConfigFromFlags(ctx, kubeconfigpath)
	if err != nil {
		return fmt.Errorf("error occurred while build rest config: %s", err)
	}

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

func WaitCRDDeleted(kubeconfigpath string, ctx string, name string) error {
	restcfg, err := buildConfigFromFlags(ctx, kubeconfigpath)
	if err != nil {
		return fmt.Errorf("error occurred while build rest config: %s", err)
	}

	client, err := apiextensionsclient.NewForConfig(restcfg)
	if err != nil {
		return err
	}

	return wait.PollUntilContextCancel(context.TODO(), 1*time.Second, true, func(ctx context.Context) (bool, error) {
		_, err = client.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	})
}

// buildConfigFromFlags build rest config for specified context in the kubeconfigfile.
func buildConfigFromFlags(context, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}
