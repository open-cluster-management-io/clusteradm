// Copyright Contributors to the Open Cluster Management project
package genericclioptions

import (
	"slices"
	"testing"

	"k8s.io/component-base/featuregate"
	ocmfeature "open-cluster-management.io/api/feature"
	operatorv1 "open-cluster-management.io/api/operator/v1"
)

func TestConvertToFeatureGateAPI(t *testing.T) {
	tests := []struct {
		name               string
		featureGates       func() featuregate.MutableFeatureGate
		defaultFeatureGate map[featuregate.Feature]featuregate.FeatureSpec
		expected           []operatorv1.FeatureGate
	}{
		{
			name: "disable default feature gate",
			featureGates: func() featuregate.MutableFeatureGate {
				fg := featuregate.NewFeatureGate()
				_ = fg.Add(map[featuregate.Feature]featuregate.FeatureSpec{
					"AddonManagement": {Default: false},
				})
				return fg
			},
			defaultFeatureGate: ocmfeature.DefaultHubAddonManagerFeatureGates,
			expected: []operatorv1.FeatureGate{
				{Feature: "AddonManagement", Mode: operatorv1.FeatureGateModeTypeDisable},
			},
		},
		{
			name: "enable default feature gate",
			featureGates: func() featuregate.MutableFeatureGate {
				fg := featuregate.NewFeatureGate()
				_ = fg.Add(map[featuregate.Feature]featuregate.FeatureSpec{
					"AddonManagement": {Default: true},
				})
				return fg
			},
			defaultFeatureGate: ocmfeature.DefaultHubAddonManagerFeatureGates,
			expected: []operatorv1.FeatureGate{
				{Feature: "AddonManagement", Mode: operatorv1.FeatureGateModeTypeEnable},
			},
		},
		{
			name: "enable non-default feature gate",
			featureGates: func() featuregate.MutableFeatureGate {
				fg := featuregate.NewFeatureGate()
				_ = fg.Add(map[featuregate.Feature]featuregate.FeatureSpec{
					"ManifestWorkReplicaSet": {Default: true},
				})
				return fg
			},
			defaultFeatureGate: ocmfeature.DefaultHubWorkFeatureGates,
			expected: []operatorv1.FeatureGate{
				{Feature: "ManifestWorkReplicaSet", Mode: operatorv1.FeatureGateModeTypeEnable},
			},
		},
		{
			name: "enable non-default feature gate, ensure default feature gates remain enabled",
			featureGates: func() featuregate.MutableFeatureGate {
				fg := featuregate.NewFeatureGate()
				_ = fg.Add(map[featuregate.Feature]featuregate.FeatureSpec{
					"MultipleHubs": {Default: true},
				})
				return fg
			},
			defaultFeatureGate: ocmfeature.DefaultSpokeRegistrationFeatureGates,
			expected: []operatorv1.FeatureGate{
				{Feature: "AddonManagement", Mode: operatorv1.FeatureGateModeTypeEnable},
				{Feature: "ClusterClaim", Mode: operatorv1.FeatureGateModeTypeEnable},
				{Feature: "MultipleHubs", Mode: operatorv1.FeatureGateModeTypeEnable},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ConvertToFeatureGateAPI(tt.featureGates(), tt.defaultFeatureGate)
			slices.SortFunc(actual, func(i, j operatorv1.FeatureGate) int {
				if i.Feature < j.Feature {
					return -1
				} else if i.Feature > j.Feature {
					return 1
				}
				return 0
			})
			if !slices.Equal(actual, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, actual)
			}
		})
	}
}
