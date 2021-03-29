// Copyright Contributors to the Open Cluster Management project

package applierscenarios

import (
	"embed"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

//go:embed embed_test
var files embed.FS
var testDir = filepath.Join("..", "..", "test", "unit")
var testDirTmp = filepath.Join(testDir, "tmp")

func TestResources_Asset(t *testing.T) {
	asset := "embed_test/detach/hub/managed_cluster_cr.yaml"
	basset, errFile := ioutil.ReadFile(asset)
	if errFile != nil {
		t.Error(errFile)
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		b       *ApplierScenarioResourcesReader
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Existing asset",
			b:    NewApplierScenarioResourcesReader(&files),
			args: args{
				name: "embed_test/detach/hub/managed_cluster_cr.yaml",
			},
			want:    basset,
			wantErr: false,
		},
		{
			name: "Not found asset",
			b:    NewApplierScenarioResourcesReader(&files),
			args: args{
				name: "hello",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.b.Asset(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resources.Asset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resources.Asset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResources_AssetNames(t *testing.T) {
	tests := []struct {
		name    string
		b       *ApplierScenarioResourcesReader
		wantErr bool
	}{
		{
			name:    "Existing asset",
			b:       NewApplierScenarioResourcesReader(&files),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewApplierScenarioResourcesReader(&files)
			got, err := b.AssetNames()
			if (err != nil) != tt.wantErr {
				t.Errorf("Resources.AssetNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// t.Logf(strings.Join(got, "\n"))
			//Check if all files returned by assetNames exists
			for _, a := range got {
				if _, err := os.Stat(a); os.IsNotExist(err) {
					t.Errorf("File not found: %s", a)
				}
			}
			//Check if all files in resources are in AssetNames except resources.go and resources_test.go
			err = filepath.Walk("./embed_test",
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() {
						// t.Logf("Check file: %s", path)
						found := false
						for _, a := range got {
							// t.Logf("Check %s == %s", a, path)
							if a == path {
								found = true
								break
							}
						}
						if !found {
							return fmt.Errorf("AssetNames does not contain file: %s", path)
						}
					}
					return nil
				})
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestResources_ToJSON(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		b       *ApplierScenarioResourcesReader
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Good yaml",
			b:    NewApplierScenarioResourcesReader(&files),
			args: args{
				b: []byte("greetings: hello"),
			},
			want:    []byte("{\"greetings\":\"hello\"}"),
			wantErr: false,
		},
		{
			name: "Bad yaml",
			b:    &ApplierScenarioResourcesReader{},
			args: args{
				b: []byte(": hello"),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewApplierScenarioResourcesReader(&files)
			got, err := b.ToJSON(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resources.ToJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resources.ToJSON() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNewResourcesReader(t *testing.T) {
	tests := []struct {
		name string
		want *ApplierScenarioResourcesReader
	}{
		{
			name: "Create",
			want: &ApplierScenarioResourcesReader{
				files: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewApplierScenarioResourcesReader(nil); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewResourcesReader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResources_ExtractAssets(t *testing.T) {
	type args struct {
		prefix string
		dir    string
	}
	tests := []struct {
		name    string
		r       *ApplierScenarioResourcesReader
		args    args
		wantErr bool
	}{
		{
			name: "Existing prefix",
			args: args{
				prefix: "embed_test/detach/hub",
				dir:    filepath.Join(testDirTmp, "exist_prefix"),
			},
			wantErr: false,
		},
		{
			name: "Existing name",
			args: args{
				prefix: "embed_test/detach/hub/managed_cluster_cr.yaml",
				dir:    filepath.Join(testDirTmp, "exist_name"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewApplierScenarioResourcesReader(&files)
			os.RemoveAll(tt.args.dir)
			if err := r.ExtractAssets(tt.args.prefix, tt.args.dir); (err != nil) != tt.wantErr {
				t.Errorf("Resources.ExtractAssets() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
