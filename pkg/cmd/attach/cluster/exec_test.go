// Copyright Contributors to the Open Cluster Management project
package cluster

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	appliercmd "github.com/open-cluster-management/applier/pkg/applier/cmd"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	crclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testDir = filepath.Join("test", "unit")

func TestOptions_complete(t *testing.T) {
	type fields struct {
		applierScenariosOptions *applierscenarios.ApplierScenariosOptions
		values                  map[string]interface{}
		clusterName             string
		clusterServer           string
		clusterToken            string
		clusterKubeConfig       string
		importFile              string
	}
	type args struct {
		cmd  *cobra.Command
		args []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Failed, bad valuesPath",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{
					ValuesPath: "bad-values-path.yaml",
				},
			},
			wantErr: true,
		},
		{
			name: "Failed, empty values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{
					ValuesPath: filepath.Join(testDir, "values-empty.yaml"),
				},
			},
			wantErr: true,
		},
		{
			name: "Sucess, not replacing values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{
					ValuesPath: filepath.Join(testDir, "values-with-data.yaml"),
				},
			},
			wantErr: false,
		},
		{
			name: "Sucess, replacing values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{
					ValuesPath: filepath.Join(testDir, "values-with-data.yaml"),
				},
				clusterServer:     "overwriteServer",
				clusterToken:      "overwriteToken",
				clusterKubeConfig: "overwriteKubeConfig",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				applierScenariosOptions: tt.fields.applierScenariosOptions,
				values:                  tt.fields.values,
				clusterName:             tt.fields.clusterName,
				clusterServer:           tt.fields.clusterServer,
				clusterToken:            tt.fields.clusterToken,
				clusterKubeConfig:       tt.fields.clusterKubeConfig,
				importFile:              tt.fields.importFile,
			}
			if err := o.complete(tt.args.cmd, tt.args.args); (err != nil) != tt.wantErr {
				t.Errorf("Options.complete() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				imc, ok := o.values["managedCluster"]
				if !ok || imc == nil {
					t.Errorf("missing managedCluster")
				}
				mc := imc.(map[string]interface{})

				if tt.name == "Sucess, replacing values" {
					if mc["kubeConfig"] != o.clusterKubeConfig {
						t.Errorf("Expect %s got %s", o.clusterKubeConfig, mc["kubeConfig"])
					}
					if mc["server"] != o.clusterServer {
						t.Errorf("Expect %s got %s", o.clusterServer, mc["server"])
					}
					if mc["token"] != o.clusterToken {
						t.Errorf("Expect %s got %s", o.clusterToken, mc["token"])
					}
				}
				if tt.name == "Sucess, not replacing values" {
					if mc["kubeConfig"] != "myKubeConfig" {
						t.Errorf("Expect %s got %s", "myKubeConfig", mc["kubeConfig"])
					}
					if mc["server"] != "myServer" {
						t.Errorf("Expect %s got %s", "myServer", mc["server"])
					}
					if mc["token"] != "myToken" {
						t.Errorf("Expect %s got %s", "myToken", mc["token"])
					}
				}
			}
		})
	}
}

func TestAttachClusterOptions_ValidateWithClient(t *testing.T) {
	client := crclientfake.NewFakeClient()
	type fields struct {
		applierScenariosOptions *applierscenarios.ApplierScenariosOptions
		values                  map[string]interface{}
		clusterName             string
		clusterServer           string
		clusterToken            string
		clusterKubeConfig       string
		importFile              string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Success local-cluster, all info in values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name": "local-cluster",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Failed local-cluster, cluster name empty",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name": "",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Failed local-cluster, cluster name missing",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values:                  map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "Success non-local-cluster, overrite cluster-name with local-cluster",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name": "test-cluster",
					},
				},

				clusterName: "local-cluster",
			},
			wantErr: false,
		},
		{
			name: "Failed non-local-cluster, missing kubeconfig or token/server",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name": "cluster-test",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Success non-local-cluster, with kubeconfig",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name": "cluster-test",
					},
				},

				clusterKubeConfig: "fake-config",
			},
			wantErr: false,
		},
		{
			name: "Success non-local-cluster, with token/server",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name": "cluster-test",
					},
				},

				clusterToken:  "fake-token",
				clusterServer: "fake-server",
			},
			wantErr: false,
		},
		{
			name: "Failed non-local-cluster, with token no server",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name": "cluster-test",
					},
				},
				clusterToken: "fake-token",
			},
			wantErr: true,
		},
		{
			name: "Failed non-local-cluster, with server no token",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name": "cluster-test",
					},
				},
				clusterServer: "fake-server",
			},
			wantErr: true,
		},
		{
			name: "Failed non-local-cluster, with kubeconfig and token/server",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name": "cluster-test",
					},
				},
				clusterKubeConfig: "fake-config",
				clusterToken:      "fake-token",
				clusterServer:     "fake-server",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				applierScenariosOptions: tt.fields.applierScenariosOptions,
				values:                  tt.fields.values,
				clusterName:             tt.fields.clusterName,
				clusterServer:           tt.fields.clusterServer,
				clusterToken:            tt.fields.clusterToken,
				clusterKubeConfig:       tt.fields.clusterKubeConfig,
				importFile:              tt.fields.importFile,
			}
			if err := o.validateWithClient(client); (err != nil) != tt.wantErr {
				t.Errorf("AttachClusterOptions.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOptions_runWithClient(t *testing.T) {
	dir, err := ioutil.TempDir(testDir, "tmp")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)
	generatedImportFileName := filepath.Join(dir, "import.yaml")
	resultImportFileName := filepath.Join(testDir, "import_result.yaml")
	os.Remove(generatedImportFileName)
	importSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-import",
			Namespace: "test",
		},
		Data: map[string][]byte{
			"crds.yaml":   []byte("crds: mycrds"),
			"import.yaml": []byte("import: myimport"),
		},
	}
	client := crclientfake.NewFakeClient(&importSecret)
	values, err := appliercmd.ConvertValuesFileToValuesMap(filepath.Join(testDir, "values-with-data.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	type fields struct {
		applierScenariosOptions *applierscenarios.ApplierScenariosOptions
		values                  map[string]interface{}
		clusterName             string
		clusterServer           string
		clusterToken            string
		clusterKubeConfig       string
		importFile              string
	}
	type args struct {
		client crclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{
					//Had to set to 1 sec otherwise test timeout is reached (30s)
					Timeout: 1,
				},
				values:      values,
				importFile:  generatedImportFileName,
				clusterName: "test",
			},
			args: args{
				client: client,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				applierScenariosOptions: tt.fields.applierScenariosOptions,
				values:                  tt.fields.values,
				clusterName:             tt.fields.clusterName,
				clusterServer:           tt.fields.clusterServer,
				clusterToken:            tt.fields.clusterToken,
				clusterKubeConfig:       tt.fields.clusterKubeConfig,
				importFile:              tt.fields.importFile,
			}
			if err := o.runWithClient(tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("Options.runWithClient() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				ns := &corev1.Namespace{}
				err = client.Get(context.TODO(),
					crclient.ObjectKey{
						Name: "test",
					},
					ns)
				if err != nil {
					t.Error(err)
				}
				//TO DO add test on exists managedcluster
				generatedImportFile, err := ioutil.ReadFile(generatedImportFileName)
				if err != nil {
					t.Error(err)
				}
				resultImportFile, err := ioutil.ReadFile(resultImportFileName)
				if err != nil {
					t.Error(err)
				}
				if !bytes.Equal(generatedImportFile, resultImportFile) {
					t.Errorf("expected import file doesn't match expected got: \n%s\n expected:\n%s\n",
						string(generatedImportFile),
						string(resultImportFile))
				}
			}
		})
	}
}
