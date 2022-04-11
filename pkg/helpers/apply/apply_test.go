// Copyright Contributors to the Open Cluster Management project
package apply

import (
	"reflect"
	"testing"

	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"open-cluster-management.io/clusteradm/pkg/helpers/asset"
	"open-cluster-management.io/clusteradm/test/unit/resources/scenario"
)

func TestMustTemplateAsset(t *testing.T) {
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
	ab := NewApplierBuilder()
	a := ab.Build()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := a.MustTemplateAsset(tt.args.reader, tt.args.values, tt.args.headerFile, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("MustTemplateAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MustTemplateAsset() = \n<EOF>%v</EOF>\n, want \n<EOF>%v</EOF>", string(got), string(tt.want))
			}
		})
	}
}

func TestApplier_Default_GetCache(t *testing.T) {
	tests := []struct {
		name string
		want resourceapply.ResourceCache
	}{
		{
			name: "get default cache",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applierBuilder := NewApplierBuilder()
			applier := applierBuilder.Build()

			if got := applier.GetCache(); got == nil {
				t.Errorf("Applier.GetCache() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplier_GetCache(t *testing.T) {
	cache := NewResourceCache()
	tests := []struct {
		name string
		want resourceapply.ResourceCache
	}{
		{
			name: "get default cache",
			want: cache,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applierBuilder := NewApplierBuilder()
			applier := applierBuilder.WithCache(cache).Build()

			if got := applier.GetCache(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Applier.GetCache() = %v, want %v", got, tt.want)
			}
		})
	}
}
