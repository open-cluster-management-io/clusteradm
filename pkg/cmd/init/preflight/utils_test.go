// Copyright Contributors to the Open Cluster Management project
package preflight

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	testinghelper "open-cluster-management.io/clusteradm/pkg/helpers/testing"
)

func TestCreateOrUpdateConfigMap(t *testing.T) {
	type args struct {
		cm     *corev1.ConfigMap
		object []runtime.Object
	}
	tests := []struct {
		name        string
		args        args
		actionIndex int
		action      string
		wantErr     bool
	}{
		{
			name: "Create",
			args: args{
				cm:     newConfigMap(BootstrapConfigMap, metav1.NamespacePublic, newKubeConfig()),
				object: []runtime.Object{},
			},
			actionIndex: 0,
			action:      "create",
			wantErr:     false,
		},
		{
			name: "Update",
			args: args{
				cm:     newConfigMap(BootstrapConfigMap, metav1.NamespacePublic, newKubeConfig()),
				object: []runtime.Object{newConfigMap(BootstrapConfigMap, metav1.NamespacePublic, nil)},
			},
			actionIndex: 1,
			action:      "update",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fakekube.NewSimpleClientset(tt.args.object...)
			if err := CreateOrUpdateConfigMap(client, tt.args.cm); (err != nil) != tt.wantErr {
				t.Errorf("CreateOrUpdateConfigMap() error = %v, wantErr %v", err, tt.wantErr)
			}
			testinghelper.AssertAction(t, client.Actions()[tt.actionIndex], tt.action)
		})
	}
}

func newConfigMap(name, namespace string, kubeconfig *clientcmdapi.Config) *corev1.ConfigMap {
	var data string
	if kubeconfig != nil {
		_ = clientcmdapi.FlattenConfig(kubeconfig)
		b, _ := clientcmd.Write(*kubeconfig)
		data = string(b)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"kubeconfig": data,
		},
	}
}

func newKubeConfig() *clientcmdapi.Config {
	return &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"": newCluster(),
		},
	}
}

func newCluster() *clientcmdapi.Cluster {
	return &clientcmdapi.Cluster{
		Server: "https://localhost",
	}
}
