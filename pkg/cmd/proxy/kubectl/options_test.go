// Copyright Contributors to the Open Cluster Management project
package kubectl

import (
	"testing"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

func Test_Options_validate(t *testing.T) {
	tests := []struct {
		name                  string
		cluster               string
		clusters              []string
		managedServiceAccount string
		wantErr               bool
		wantErrMsg            string
	}{
		{
			name:                  "valid single cluster",
			cluster:               "cluster1",
			managedServiceAccount: "sa1",
			wantErr:               false,
		},
		{
			name:                  "clusters flag rejected",
			cluster:               "cluster1",
			clusters:              []string{"cluster1", "cluster2"},
			managedServiceAccount: "sa1",
			wantErr:               true,
			wantErrMsg:            "proxy kubectl only supports a single cluster, use --cluster instead of --clusters",
		},
		{
			name:                  "clusters flag alone rejected",
			clusters:              []string{"cluster1", "cluster2"},
			managedServiceAccount: "sa1",
			wantErr:               true,
			wantErrMsg:            "proxy kubectl only supports a single cluster, use --cluster instead of --clusters",
		},
		{
			name:                  "missing cluster",
			managedServiceAccount: "sa1",
			wantErr:               true,
			wantErrMsg:            "--cluster is required",
		},
		{
			name:       "missing managedServiceAccount",
			cluster:    "cluster1",
			wantErr:    true,
			wantErrMsg: "managedServiceAccount is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				ClusterOption:         genericclioptionsclusteradm.NewClusterOption(),
				managedServiceAccount: tt.managedServiceAccount,
			}
			o.ClusterOption.Cluster = tt.cluster
			o.ClusterOption.Clusters = tt.clusters

			err := o.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && err.Error() != tt.wantErrMsg {
				t.Errorf("validate() error message = %q, want %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}
