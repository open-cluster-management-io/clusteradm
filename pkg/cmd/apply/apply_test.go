// Copyright Contributors to the Open Cluster Management project

package apply

import (
	"reflect"
	"testing"
)

func TestConvertValuesFileToValuesMap(t *testing.T) {
	type args struct {
		path   string
		prefix string
	}
	tests := []struct {
		name       string
		args       args
		wantValues map[string]interface{}
		wantErr    bool
	}{
		{
			name: "Succeed, no prefix",
			args: args{
				path: "../../../test/unit/resources/values.yaml",
			},
			wantValues: map[string]interface{}{
				"managedCluster": map[string]interface{}{
					"name": "test-cluster",
				},
			},
			wantErr: false,
		},
		{
			name: "Succeed, with prefix",
			args: args{
				path:   "../../../test/unit/resources/values.yaml",
				prefix: "cluster",
			},
			wantValues: map[string]interface{}{
				"cluster": map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name": "test-cluster",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Failed, wrong path",
			args: args{
				path:   "fake-path",
				prefix: "cluster",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValues, err := ConvertValuesFileToValuesMap(tt.args.path, tt.args.prefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertValuesFileToValuesMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotValues, tt.wantValues) {
				t.Errorf("ConvertValuesFileToValuesMap() = %v, want %v", gotValues, tt.wantValues)
			}
		})
	}
}
