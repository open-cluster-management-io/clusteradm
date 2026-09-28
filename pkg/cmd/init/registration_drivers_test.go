// Copyright Contributors to the Open Cluster Management project
package init

import (
	"reflect"
	"testing"

	operatorv1 "open-cluster-management.io/api/operator/v1"
)

func TestGetRegistrationDriversAzure(t *testing.T) {
	cases := []struct {
		name     string
		opts     *Options
		expected []operatorv1.RegistrationDriverHub
	}{
		{
			name: "azure without auto-approval patterns",
			opts: &Options{registrationDrivers: []string{"csr", "azure"}},
			expected: []operatorv1.RegistrationDriverHub{
				{AuthType: operatorv1.CSRAuthType},
				{AuthType: operatorv1.AzureAuthType},
			},
		},
		{
			name: "azure with auto-approval patterns",
			opts: &Options{
				registrationDrivers:               []string{"azure"},
				autoApprovedAzureIdentityPatterns: []string{"^11111111-1111-1111-1111-111111111111$"},
			},
			expected: []operatorv1.RegistrationDriverHub{
				{
					AuthType: operatorv1.AzureAuthType,
					Azure: &operatorv1.AzureConfig{
						AutoApprovedIdentityPatterns: []string{"^11111111-1111-1111-1111-111111111111$"},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := getRegistrationDrivers(c.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, c.expected) {
				t.Errorf("expected %+v, got %+v", c.expected, got)
			}
		})
	}
}
