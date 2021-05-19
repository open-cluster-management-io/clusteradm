// Copyright Contributors to the Open Cluster Management project
package cluster

import (
	"reflect"
	"testing"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func Test_newOptions(t *testing.T) {
	type args struct {
		streams genericclioptions.IOStreams
	}
	tests := []struct {
		name string
		args args
		want *Options
	}{
		{
			name: "success",
			args: args{
				streams: genericclioptions.IOStreams{},
			},
			want: &Options{
				applierScenariosOptions: applierscenarios.NewApplierScenariosOptions(genericclioptions.IOStreams{}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newOptions(tt.args.streams); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
