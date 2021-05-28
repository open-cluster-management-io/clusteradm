// Copyright Contributors to the Open Cluster Management project

package helpers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/ghodss/yaml"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func GetControllerRuntimeClientFromFlags(configFlags *genericclioptions.ConfigFlags) (client crclient.Client, err error) {
	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	config.QPS = 20
	return crclient.New(config, crclient.Options{})
}

func GetAPIServer(client crclient.Client) (string, error) {
	config, err := getClusterInfoKubeConfig(client)
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

func getClusterInfoKubeConfig(client crclient.Client) (*clientcmdapiv1.Config, error) {
	cm, err := getClusterInfo(client)
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

func getClusterInfo(client crclient.Client) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	err := client.Get(context.TODO(), crclient.ObjectKey{Namespace: "kube-public", Name: "cluster-info"}, cm)
	if err != nil {
		return nil, err
	}
	return cm, nil
}
