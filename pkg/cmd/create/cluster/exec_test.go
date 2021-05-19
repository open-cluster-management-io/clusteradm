// Copyright Contributors to the Open Cluster Management project
package cluster

import (
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
		clusterName             string
		cloud                   string
		values                  map[string]interface{}
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
			name: "Failed, empty values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{
					ValuesPath: filepath.Join(testDir, "values-empty.yaml"),
				},
			},
			wantErr: true,
		},
		{
			name: "Sucess, with values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{
					ValuesPath: filepath.Join(testDir, "values-fake-aws.yaml"),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				applierScenariosOptions: tt.fields.applierScenariosOptions,
				clusterName:             tt.fields.clusterName,
				cloud:                   tt.fields.cloud,
				values:                  tt.fields.values,
			}
			if err := o.complete(tt.args.cmd, tt.args.args); (err != nil) != tt.wantErr {
				t.Errorf("Options.complete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOptions_validate(t *testing.T) {
	type fields struct {
		applierScenariosOptions *applierscenarios.ApplierScenariosOptions
		clusterName             string
		cloud                   string
		values                  map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Success AWS all info in values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name":  "test",
						"cloud": "aws",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Success Azure all info in values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name":  "test",
						"cloud": "azure",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Success GCP all info in values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name":  "test",
						"cloud": "gcp",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Success VSphere all info in values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name":  "test",
						"cloud": "vsphere",
					},
				},
			},
			wantErr: false,
		},
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
			name: "Failed managedCluster missing",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values:                  map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "Failed name missing",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"cloud": "vsphere",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Failed name empty",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name":  "",
						"cloud": "vsphere",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Failed cloud missing",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name": "test",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Failed cloud enpty",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name":  "test",
						"cloud": "",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Success replace clusterName",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name":  "test",
						"cloud": "aws",
					},
				},
				clusterName: "test2",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				applierScenariosOptions: tt.fields.applierScenariosOptions,
				clusterName:             tt.fields.clusterName,
				cloud:                   tt.fields.cloud,
				values:                  tt.fields.values,
			}
			if err := o.validate(); (err != nil) != tt.wantErr {
				t.Errorf("Options.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.name == "Success replace clusterName" {
				if imc, ok := o.values["managedCluster"]; ok {
					mc := imc.(map[string]interface{})
					if icn, ok := mc["name"]; ok {
						cm := icn.(string)
						if cm != "test2" {
							t.Errorf("got %s and expected %s", tt.fields.clusterName, cm)
						}
					} else {
						t.Error("name not found")
					}
				} else {
					t.Error("managedCluster not found")
				}
			}
		})
	}
}

func TestOptions_runWithClient(t *testing.T) {
	pullSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pull-secret",
			Namespace: "openshift-config",
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte("crds: mycrds"),
		},
	}
	client := crclientfake.NewFakeClient(&pullSecret)
	valuesAWS, err := appliercmd.ConvertValuesFileToValuesMap(filepath.Join(testDir, "values-fake-aws.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	type fields struct {
		applierScenariosOptions *applierscenarios.ApplierScenariosOptions
		clusterName             string
		cloud                   string
		values                  map[string]interface{}
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
				values: valuesAWS,
				cloud:  "aws",
			},
			args: args{
				client: client,
			},
			wantErr: false,
		},
		{
			name: "Failed no pullSecret",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{
					//Had to set to 1 sec otherwise test timeout is reached (30s)
					Timeout: 1,
				},
				values: valuesAWS,
				cloud:  "aws",
			},
			args: args{
				client: crclientfake.NewFakeClient(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				applierScenariosOptions: tt.fields.applierScenariosOptions,
				clusterName:             tt.fields.clusterName,
				cloud:                   tt.fields.cloud,
				values:                  tt.fields.values,
			}
			if err := o.runWithClient(tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("Options.runWithClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
