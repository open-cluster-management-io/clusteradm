// Copyright Contributors to the Open Cluster Management project
package placement

import (
	"k8s.io/apimachinery/pkg/api/equality"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	"testing"
)

func TestParsePrioritizer(t *testing.T) {
	cases := []struct {
		name             string
		prioritizers     string
		expectErr        bool
		expectProritizer *clusterv1beta1.PrioritizerConfig
	}{
		{
			name:      "empty string",
			expectErr: true,
		},
		{
			name:         "builtin with wrong type",
			prioritizers: "BuiltIn:Balance",
			expectErr:    true,
		},
		{
			name:         "builtin",
			prioritizers: "BuiltIn:Balance:4",
			expectErr:    false,
			expectProritizer: &clusterv1beta1.PrioritizerConfig{
				ScoreCoordinate: &clusterv1beta1.ScoreCoordinate{
					Type:    clusterv1beta1.ScoreCoordinateTypeBuiltIn,
					BuiltIn: "Balance",
				},
				Weight: 4,
			},
		},
		{
			name:         "addon with wrong type",
			prioritizers: "AddOn:cpu:usage:2:2",
			expectErr:    true,
		},
		{
			name:         "addon",
			prioritizers: "AddOn:cpu:usage:2",
			expectErr:    false,
			expectProritizer: &clusterv1beta1.PrioritizerConfig{
				ScoreCoordinate: &clusterv1beta1.ScoreCoordinate{
					Type: clusterv1beta1.ScoreCoordinateTypeAddOn,
					AddOn: &clusterv1beta1.AddOnScore{
						ResourceName: "cpu",
						ScoreName:    "usage",
					},
				},
				Weight: 2,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual, err := parsePrioritizer(c.prioritizers)
			if err != nil && !c.expectErr {
				t.Errorf("should not have error, but got %v", err)
			}
			if err == nil && c.expectErr {
				t.Errorf("should return err")
			}
			if !equality.Semantic.DeepEqual(actual, c.expectProritizer) {
				t.Errorf("expected priotizier not correct, actual %v", actual)
			}
		})
	}
}
