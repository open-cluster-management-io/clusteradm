// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package bindata

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBindata_Asset(t *testing.T) {
	asset := "scenarios/attachdetach/hub/managedcluster_cr.yaml"
	basset, errFile := ioutil.ReadFile(filepath.Join("../../resources", asset))
	if errFile != nil {
		t.Error(errFile)
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		b       *Bindata
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Existing asset",
			b:    &Bindata{},
			args: args{
				name: "scenarios/attachdetach/hub/managedcluster_cr.yaml",
			},
			want:    basset,
			wantErr: false,
		},
		// {
		// 	name: "Existing asset root empty",
		// 	b: &Bindata{
		// 		root: "",
		// 	},
		// 	args: args{
		// 		name: "scenarios/attachdetach/hub/managedcluster_cr.yaml",
		// 	},
		// 	want:    basset,
		// 	wantErr: false,
		// },
		// {
		// 	name: "Existing asset root fullpath",
		// 	b: &Bindata{
		// 		root: "scenarios/attachdetach/hub/managedcluster_cr.yaml",
		// 	},
		// 	args: args{
		// 		name: "",
		// 	},
		// 	want:    basset,
		// 	wantErr: false,
		// },
		// {
		// 	name: "Existing asset sub",
		// 	b: &Bindata{
		// 		root: "scenarios/attachdetach/hub",
		// 	},
		// 	args: args{
		// 		name: "managedcluster_cr.yaml",
		// 	},
		// 	want:    basset,
		// 	wantErr: false,
		// },
		// {
		// 	name: "Not found asset",
		// 	b:    &Bindata{},
		// 	args: args{
		// 		name: "hello",
		// 	},
		// 	want:    nil,
		// 	wantErr: true,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.b.Asset(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bindata.Asset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Bindata.Asset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBindata_AssetNames(t *testing.T) {
	tests := []struct {
		name    string
		b       *Bindata
		wantErr bool
	}{
		{
			name:    "Existing asset",
			b:       &Bindata{},
			wantErr: false,
		},
		// {
		// 	name: "Existing single asset",
		// 	b: &Bindata{
		// 		root: "scenarios/createdestroy/hub/aws",
		// 	},
		// 	wantErr: false,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bindata{}
			got, err := b.AssetNames()
			if (err != nil) != tt.wantErr {
				t.Errorf("Bindata.AssetNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) == 0 {
				t.Errorf("Bindata.AssetNames() len must be not zero")
			}
			// if got[1] != "install_config.yaml" && b.root != "" {
			// 	t.Errorf("Bindata.AssetNames() len must return install_config.yaml but got %v", got)
			// }
		})
	}
}

func TestBindata_ToJSON(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		b       *Bindata
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Good yaml",
			b:    &Bindata{},
			args: args{
				b: []byte("greetings: hello"),
			},
			want:    []byte("{\"greetings\":\"hello\"}"),
			wantErr: false,
		},
		{
			name: "Bad yaml",
			b:    &Bindata{},
			args: args{
				b: []byte(": hello"),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bindata{}
			got, err := b.ToJSON(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bindata.ToJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Bindata.ToJSON() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNewBindataReader(t *testing.T) {
	tests := []struct {
		name string
		want *Bindata
	}{
		{
			name: "Create",
			want: &Bindata{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewBindataReader(""); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBindataReader() = %v, want %v", got, tt.want)
			}
		})
	}
}
