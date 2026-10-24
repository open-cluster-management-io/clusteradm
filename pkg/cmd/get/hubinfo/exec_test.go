// Copyright Contributors to the Open Cluster Management project
package hubinfo

import (
	"testing"

	v1 "open-cluster-management.io/api/operator/v1"
)

func TestWorkControllerEnabled(t *testing.T) {
	cases := []struct {
		name       string
		workConfig *v1.WorkConfiguration
		want       bool
	}{
		{
			name:       "no work configuration uses the ManifestWorkReplicaSet default",
			workConfig: nil,
			want:       true,
		},
		{
			name:       "work configuration without feature gates uses the default",
			workConfig: &v1.WorkConfiguration{},
			want:       true,
		},
		{
			name: "ManifestWorkReplicaSet explicitly enabled",
			workConfig: &v1.WorkConfiguration{FeatureGates: []v1.FeatureGate{
				{Feature: "ManifestWorkReplicaSet", Mode: v1.FeatureGateModeTypeEnable},
			}},
			want: true,
		},
		{
			name: "ManifestWorkReplicaSet disabled",
			workConfig: &v1.WorkConfiguration{FeatureGates: []v1.FeatureGate{
				{Feature: "ManifestWorkReplicaSet", Mode: v1.FeatureGateModeTypeDisable},
			}},
			want: false,
		},
		{
			name: "ManifestWorkReplicaSet disabled but CleanUpCompletedManifestWork enabled",
			workConfig: &v1.WorkConfiguration{FeatureGates: []v1.FeatureGate{
				{Feature: "ManifestWorkReplicaSet", Mode: v1.FeatureGateModeTypeDisable},
				{Feature: "CleanUpCompletedManifestWork", Mode: v1.FeatureGateModeTypeEnable},
			}},
			want: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmgr := &v1.ClusterManager{Spec: v1.ClusterManagerSpec{WorkConfiguration: c.workConfig}}
			if got := workControllerEnabled(cmgr); got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}
}
