// Copyright Contributors to the Open Cluster Management project

package helpers

import (
	"context"
	"fmt"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/ghodss/yaml"
	"k8s.io/client-go/kubernetes"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/client-go/util/retry"
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
