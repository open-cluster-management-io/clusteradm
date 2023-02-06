// Copyright Contributors to the Open Cluster Management project
package preflight

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd/api"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	testinghelper "open-cluster-management.io/clusteradm/pkg/helpers/testing"
)

var (
	currentContext     = "ocm-hub"
	kubeconfigFilePath = "testdata/kubeconfig"
)

func Test_loadCurrentCluster(t *testing.T) {
	type args struct {
		context            string
		kubeConfigFilePath string
	}
	tests := []struct {
		name    string
		args    args
		want    *api.Cluster
		wantErr bool
	}{
		{
			name: "load",
			args: args{
				context:            currentContext,
				kubeConfigFilePath: kubeconfigFilePath,
			},
			want: &api.Cluster{
				LocationOfOrigin:         kubeconfigFilePath,
				Server:                   "https://localhost:8443",
				TLSServerName:            "",
				InsecureSkipTLSVerify:    false,
				CertificateAuthority:     "",
				CertificateAuthorityData: nil,
				ProxyURL:                 "",
				Extensions:               make(map[string]runtime.Object),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadCurrentCluster(tt.args.context, tt.args.kubeConfigFilePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadCurrentCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadCurrentCluster() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkServer(t *testing.T) {
	tests := []struct {
		name         string
		server       string
		wantErrList  []string
		wantWarnings []string
	}{
		{
			name:         "IP address and port",
			server:       "https://1.2.3.4:8443",
			wantErrList:  nil,
			wantWarnings: nil,
		},
		{
			name:         "no port",
			server:       "https://1.2.3.4",
			wantErrList:  nil,
			wantWarnings: nil,
		},
		{
			name:         "domain name",
			server:       "https://example.com:8443",
			wantErrList:  nil,
			wantWarnings: []string{"Hub Api Server is a domain name, maybe you should set HostAlias in klusterlet"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, errorList := checkServer(tt.server)
			if warnings != nil && !reflect.DeepEqual(warnings, tt.wantWarnings) {
				t.Errorf("check() warnings = %v, wantWarnings %v", warnings, tt.wantWarnings)
			}
			if errorList != nil && !reflect.DeepEqual(errorList, tt.wantErrList) {
				t.Errorf("check() errorList = %v, wantErrList %v", errorList, tt.wantErrList)
			}
		})
	}
}

func Test_createClusterInfo(t *testing.T) {
	type args struct {
		cluster *clientcmdapi.Cluster
		object  []runtime.Object
	}
	tests := []struct {
		name        string
		args        args
		actionIndex int
		action      string
		wantErr     bool
	}{
		{
			name: "create",
			args: args{
				cluster: newCluster(),
				object:  []runtime.Object{},
			},
			actionIndex: 0,
			action:      "create",
			wantErr:     false,
		},
		{
			name: "update",
			args: args{
				cluster: newCluster(),
				object:  []runtime.Object{newConfigMap(BootstrapConfigMap, metav1.NamespacePublic, nil)},
			},
			actionIndex: 1,
			action:      "update",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fakekube.NewSimpleClientset(tt.args.object...)
			if err := createClusterInfo(client, tt.args.cluster); (err != nil) != tt.wantErr {
				t.Errorf("createClusterInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
			testinghelper.AssertAction(t, client.Actions()[tt.actionIndex], tt.action)
		})
	}
}

func TestHubApiServerCheck_Check(t *testing.T) {
	type fields struct {
		ClusterCtx string
		ConfigPath string
	}
	tests := []struct {
		name          string
		fields        fields
		wantWarnings  []string
		wantErrorList []error
	}{
		{
			name: "empty context",
			fields: fields{
				ClusterCtx: "",
				ConfigPath: kubeconfigFilePath,
			},
			wantWarnings:  []string{"Hub Api Server is a domain name, maybe you should set HostAlias in klusterlet"},
			wantErrorList: nil,
		},
		{
			name: "no kubeconfig file",
			fields: fields{
				ClusterCtx: currentContext,
				ConfigPath: "invalid_path",
			},
			wantWarnings: nil,
			wantErrorList: []error{
				errors.New("failed to load kubeconfig file invalid_path that already exists on disk: open invalid_path: no such file or directory"),
			},
		},
		{
			name: "hub api server with domain",
			fields: fields{
				ClusterCtx: currentContext,
				ConfigPath: kubeconfigFilePath,
			},
			wantWarnings:  []string{"Hub Api Server is a domain name, maybe you should set HostAlias in klusterlet"},
			wantErrorList: nil,
		},
		{
			name: "hub api server with ip",
			fields: fields{
				ClusterCtx: currentContext,
				ConfigPath: "testdata/kubeconfig_ip",
			},
			wantWarnings:  nil,
			wantErrorList: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := HubApiServerCheck{
				ClusterCtx: tt.fields.ClusterCtx,
				ConfigPath: tt.fields.ConfigPath,
			}
			gotWarnings, gotErrorList := c.Check()
			testinghelper.AssertWarnings(t, gotWarnings, tt.wantWarnings)
			testinghelper.AssertErrors(t, gotErrorList, tt.wantErrorList)
		})
	}
}

func TestClusterInfoCheck_Check(t *testing.T) {
	type fields struct {
		Namespace    string
		ResourceName string
		ClusterCtx   string
		ConfigPath   string
		Object       []runtime.Object
	}
	tests := []struct {
		name          string
		fields        fields
		actionIndex   int
		action        string
		wantWarnings  []string
		wantErrorList []error
	}{
		{
			name: "ConfigMap existed",
			fields: fields{
				Namespace:    metav1.NamespacePublic,
				ResourceName: BootstrapConfigMap,
				ClusterCtx:   currentContext,
				ConfigPath:   kubeconfigFilePath,
				Object:       []runtime.Object{newConfigMap(BootstrapConfigMap, metav1.NamespacePublic, newKubeConfig())},
			},
			actionIndex:   0,
			action:        "get",
			wantWarnings:  nil,
			wantErrorList: nil,
		},
		{
			name: "invalid ConfigMap data",
			fields: fields{
				Namespace:    metav1.NamespacePublic,
				ResourceName: BootstrapConfigMap,
				ClusterCtx:   currentContext,
				ConfigPath:   kubeconfigFilePath,
				Object:       []runtime.Object{newConfigMap(BootstrapConfigMap, metav1.NamespacePublic, nil)},
			},
			actionIndex:   0,
			action:        "get",
			wantWarnings:  nil,
			wantErrorList: []error{errors.New("empty kubeconfig data in cluster-info")},
		},
		{
			name: "no ConfigMap existed",
			fields: fields{
				Namespace:    metav1.NamespacePublic,
				ResourceName: BootstrapConfigMap,
				ClusterCtx:   currentContext,
				ConfigPath:   kubeconfigFilePath,
				Object:       []runtime.Object{},
			},
			actionIndex:   1,
			action:        "create",
			wantWarnings:  []string{"no ConfigMap named cluster-info in the kube-public namespace, clusteradm will creates it"},
			wantErrorList: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fakekube.NewSimpleClientset(tt.fields.Object...)

			c := ClusterInfoCheck{
				Namespace:    tt.fields.Namespace,
				ResourceName: tt.fields.ResourceName,
				ClusterCtx:   tt.fields.ClusterCtx,
				ConfigPath:   tt.fields.ConfigPath,
				Client:       client,
			}
			gotWarnings, gotErrorList := c.Check()
			testinghelper.AssertAction(t, client.Actions()[tt.actionIndex], tt.action)
			testinghelper.AssertWarnings(t, gotWarnings, tt.wantWarnings)
			testinghelper.AssertErrors(t, gotErrorList, tt.wantErrorList)
		})
	}
}
