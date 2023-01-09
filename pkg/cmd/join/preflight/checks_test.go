// Copyright Contributors to the Open Cluster Management project
package preflight

import (
	"testing"

	"github.com/pkg/errors"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	testinghelper "open-cluster-management.io/clusteradm/pkg/helpers/testing"
)

func TestHubKubeconfigCheck(t *testing.T) {
	testcases := []struct {
		name          string
		config        *clientcmdapiv1.Config
		wantWarnings  []string
		wantErrorList []error
	}{
		{
			name:          "config equals nil",
			config:        nil,
			wantWarnings:  nil,
			wantErrorList: []error{errors.New("no hubconfig found")},
		},
		{
			name:          "cluster length is not 1",
			config:        &clientcmdapiv1.Config{},
			wantWarnings:  nil,
			wantErrorList: []error{errors.New("error cluster length")},
		},
		{
			name: "apiserver format is not valid",
			config: &clientcmdapiv1.Config{
				Clusters: []clientcmdapiv1.NamedCluster{
					{
						Cluster: clientcmdapiv1.Cluster{
							Server: "1.2.3.4",
						},
					},
				},
			},
			wantWarnings:  nil,
			wantErrorList: []error{errors.New("--hub-apiserver should start with http:// or https://")},
		},
		{
			name: "ca not validated",
			config: &clientcmdapiv1.Config{
				Clusters: []clientcmdapiv1.NamedCluster{
					{
						Cluster: clientcmdapiv1.Cluster{
							Server:                   "https://1.2.3.4",
							CertificateAuthorityData: nil,
						},
					},
				},
			},
			wantWarnings:  nil,
			wantErrorList: []error{errors.New("no ca detected, creating hub kubeconfig without ca")},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			c := HubKubeconfigCheck{
				Config: tc.config,
			}
			gotWarnings, gotErrorList := c.Check()
			testinghelper.AssertWarnings(t, gotWarnings, tc.wantWarnings)
			testinghelper.AssertErrors(t, gotErrorList, tc.wantErrorList)
		})
	}
}
