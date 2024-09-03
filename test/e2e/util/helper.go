// Copyright Contributors to the Open Cluster Management project
package util

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	clusterapiv1 "open-cluster-management.io/api/cluster/v1"
	"open-cluster-management.io/clusteradm/pkg/config"

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
		fmt.Printf("namespace %s still exists %v\n", ns.Name, ns.Status)
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

		if !meta.IsStatusConditionTrue(managedCluster.Status.Conditions, clusterapiv1.ManagedClusterConditionAvailable) {
			return true, nil
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
	if errors.IsNotFound(err) {
		return nil
	}
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

func ValidateImagePullSecret(kubeClient kubernetes.Interface, expectedCred string, namespace string) error {
	pullSecret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), config.ImagePullSecret, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cannot find pull secret in %v ns. %v", namespace, err)
	}
	if base64.StdEncoding.EncodeToString(pullSecret.Data[".dockerconfigjson"]) != expectedCred {
		return fmt.Errorf("unexpected .dockerconfigjson %v of pull secret in ns %v.expected:%v",
			base64.StdEncoding.EncodeToString(pullSecret.Data[".dockerconfigjson"]), namespace, expectedCred)
	}

	return nil
}

func NewTestImagePullCredentialFile(fileName string) string {
	data := `{"auths":{}}`
	_ = os.WriteFile(fileName, []byte(data), 0600)
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func CleanupTestImagePullCredentialFile(fileName string) {
	_ = os.Remove(fileName)
}
