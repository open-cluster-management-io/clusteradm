// Copyright Contributors to the Open Cluster Management project
package apply

import (
	"reflect"
	"testing"

	"open-cluster-management.io/clusteradm/pkg/helpers/asset"
	"open-cluster-management.io/clusteradm/test/unit/resources/scenario"
)

func TestMustTempalteAsset(t *testing.T) {
	type args struct {
		name       string
		headerFile string
		reader     asset.ScenarioReader
		values     interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				name:       "musttemplateasset/body.txt",
				headerFile: "",
				reader:     scenario.GetScenarioResourcesReader(),
				values:     map[string]string{"myvalue": "thevalue"},
			},
			want:    []byte("# hello\nthevalue"),
			wantErr: false,
		},
		{
			name: "success with header",
			args: args{
				name:       "musttemplateasset/body_for_header.txt",
				headerFile: "musttemplateasset/header.txt",
				reader:     scenario.GetScenarioResourcesReader(),
				values:     map[string]string{"myvalue": "thevalue"},
			},
			want:    []byte("thevalue\n\n\nhello"),
			wantErr: false,
		},
		{
			name: "fails syntax error template",
			args: args{
				name:       "musttemplateasset/body_with_syntax_error.txt",
				headerFile: "",
				reader:     scenario.GetScenarioResourcesReader(),
				values:     map[string]string{"myvalue": "thevalue"},
			},
			wantErr: true,
		},
		{
			name: "fails syntax error template with header",
			args: args{
				name:       "musttemplateasset/body_with_syntax_error.txt",
				headerFile: "musttemplateasset/header.txt",
				reader:     scenario.GetScenarioResourcesReader(),
				values:     map[string]string{"myvalue": "thevalue"},
			},
			wantErr: true,
		},
		{
			name: "fails empty",
			args: args{
				name:       "musttemplateasset/body_empty.txt",
				headerFile: "",
				reader:     scenario.GetScenarioResourcesReader(),
				values:     map[string]string{},
			},
			wantErr: true,
		},
	}
	ab := &ApplierBuilder{}
	a := ab.Build()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := a.MustTempalteAsset(tt.args.reader, tt.args.values, tt.args.headerFile, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("MustTempalteAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MustTempalteAsset() = \n<EOF>%v</EOF>\n, want \n<EOF>%v</EOF>", string(got), string(tt.want))
			}
		})
	}
}
