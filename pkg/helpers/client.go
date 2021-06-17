// Copyright Contributors to the Open Cluster Management project

package helpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/client-go/util/retry"
	"open-cluster-management.io/clusteradm/pkg/config"
)

func GetAPIServer(kubeClient kubernetes.Interface) (string, error) {
	config, err := getClusterInfoKubeConfig(kubeClient)
	if err != nil {
		return "", err
	}
	clusters := config.Clusters
	if len(clusters) != 1 {
		return "", fmt.Errorf("can not find the cluster in the cluster-info")
	}
	cluster := clusters[0].Cluster
	return cluster.Server, nil
}

func GetCACert(kubeClient kubernetes.Interface) ([]byte, error) {
	config, err := getClusterInfoKubeConfig(kubeClient)
	if err == nil {
		clusters := config.Clusters
		if len(clusters) != 1 {
			return nil, fmt.Errorf("can not find the cluster in the cluster-info")
		}
		cluster := clusters[0].Cluster
		return cluster.CertificateAuthorityData, nil
	}
	if errors.IsNotFound(err) {
		cm, err := kubeClient.CoreV1().ConfigMaps("kube-public").Get(context.TODO(), "kube-root-ca.crt", metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return []byte(cm.Data["ca.crt"]), nil
	}
	return nil, err
}

func getClusterInfoKubeConfig(kubeClient kubernetes.Interface) (*clientcmdapiv1.Config, error) {
	cm, err := kubeClient.CoreV1().ConfigMaps("kube-public").Get(context.TODO(), "cluster-info", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	config := &clientcmdapiv1.Config{}
	err = yaml.Unmarshal([]byte(cm.Data["kubeconfig"]), config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func WaitCRDToBeReady(apiExtensionsClient apiextensionsclient.Clientset, name string, b wait.Backoff) error {
	errGet := retry.OnError(b, func(err error) bool {
		if err != nil {
			fmt.Printf("Wait  for %s crd to be ready\n", name)
			return true
		}
		return false
	}, func() error {
		_, err := apiExtensionsClient.ApiextensionsV1().CustomResourceDefinitions().
			Get(context.TODO(),
				name,
				metav1.GetOptions{})
		return err
	})
	return errGet
}

func GetBootstrapSecret(
	kubeClient kubernetes.Interface) (*corev1.Secret, error) {
	sa, err := kubeClient.CoreV1().
		ServiceAccounts(config.OpenClusterManagementNamespace).
		Get(context.TODO(), config.BootstrapSAName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var secret *corev1.Secret
	for _, objectRef := range sa.Secrets {
		if objectRef.Namespace != "" && objectRef.Namespace != config.OpenClusterManagementNamespace {
			continue
		}
		prefix := config.BootstrapSAName
		if len(prefix) > 63 {
			prefix = prefix[:37]
		}
		if strings.HasPrefix(objectRef.Name, prefix) {
			secret, err = kubeClient.CoreV1().
				Secrets(config.OpenClusterManagementNamespace).
				Get(context.TODO(), objectRef.Name, metav1.GetOptions{})
			if err != nil {
				continue
			}
			if secret.Type == corev1.SecretTypeServiceAccountToken {
				break
			}
		}
	}
	if secret == nil {
		return nil, fmt.Errorf("secret with prefix %s and type %s not found in service account %s/%s",
			config.BootstrapSAName,
			corev1.SecretTypeServiceAccountToken,
			config.OpenClusterManagementNamespace,
			config.BootstrapSAName)
	}
	return secret, nil
}

func IsClusterManagerInstalled(apiExtensionsClient apiextensionsclient.Interface) (bool, error) {
	_, err := apiExtensionsClient.ApiextensionsV1().
		CustomResourceDefinitions().
		Get(context.TODO(), "clustermanagers.operator.open-cluster-management.io", metav1.GetOptions{})
	if err == nil {
		return true, nil
	}
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
	}
	return false, err
}
