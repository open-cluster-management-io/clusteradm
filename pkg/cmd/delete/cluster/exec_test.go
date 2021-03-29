// Copyright Contributors to the Open Cluster Management project
package cluster

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/open-cluster-management/applier/pkg/applier"
	appliercmd "github.com/open-cluster-management/applier/pkg/applier/cmd"
	"github.com/open-cluster-management/applier/pkg/templateprocessor"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	crclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testDir = filepath.Join("test", "unit")

func TestOptions_complete(t *testing.T) {
	type fields struct {
		applierScenariosOptions *applierscenarios.ApplierScenariosOptions
		clusterName             string
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
			name: "Sucess, with values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{
					ValuesPath: filepath.Join(testDir, "values-fake.yaml"),
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
						"name": "test",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Failed name missing",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{},
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
						"name": "",
					},
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
				values:                  tt.fields.values,
			}
			if err := o.validate(); (err != nil) != tt.wantErr {
				t.Errorf("Options.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOptions_runWithClient(t *testing.T) {
	existingMC := `apiVersion: cluster.open-cluster-management.io/v1
kind: ManagedCluster
metadata:
  labels:
    cloud: auto-detect
    vendor: auto-detect
  name: "run-with-client"
spec:
  hubAcceptsClient: true
  leaseDurationSeconds: 60
---
apiVersion: hive.openshift.io/v1
kind: ClusterDeployment
metadata:
  name: "run-with-client"
  namespace: "run-with-client"

`
	client := crclientfake.NewFakeClient()
	values, err := appliercmd.ConvertValuesFileToValuesMap(filepath.Join(testDir, "values-with-data.yaml"), "")
	if err != nil {
		t.Error(err)
	}

	reader := templateprocessor.NewYamlStringReader(existingMC, templateprocessor.KubernetesYamlsDelimiter)
	a, err := applier.NewApplier(reader, &templateprocessor.Options{}, client, nil, nil, applier.DefaultKubernetesMerger, &applier.Options{})
	if err != nil {
		t.Error(err)
	}
	err = a.CreateResources([]string{"0"}, values)
	if err != nil {
		t.Error(err)
	}
	type fields struct {
		applierScenariosOptions *applierscenarios.ApplierScenariosOptions
		clusterName             string
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
				values:      values,
				clusterName: "test",
			},
			args: args{
				client: client,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				applierScenariosOptions: tt.fields.applierScenariosOptions,
				// clusterName:             tt.fields.clusterName,
				values: tt.fields.values,
			}
			if err := o.runWithClient(tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("Options.runWithClient() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				obj := unstructured.Unstructured{}
				obj.SetKind("ManagedCluster")
				obj.SetAPIVersion("cluster.open-cluster-management.io/v1")
				err := tt.args.client.Get(context.TODO(), crclient.ObjectKey{Name: tt.fields.clusterName}, &obj)
				if err != nil && !errors.IsNotFound(err) {
					t.Error(err)
				}
				obj = unstructured.Unstructured{}
				obj.SetKind("ClusterDeployment")
				obj.SetAPIVersion("hive.openshift.io/v1")
				err = tt.args.client.Get(context.TODO(), crclient.ObjectKey{Name: tt.fields.clusterName}, &obj)
				if err != nil && !errors.IsNotFound(err) {
					t.Error(err)
				}
			}
		})
	}
}
