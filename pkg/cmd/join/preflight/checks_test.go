// Copyright Contributors to the Open Cluster Management project
package preflight

import (
	"testing"

	"github.com/pkg/errors"
	testinghelper "open-cluster-management.io/clusteradm/pkg/helpers/testing"
)

func TestKlusterletApiHostCheck_Check(t *testing.T) {
	type fields struct {
		apihost string
	}
	tests := []struct {
		name          string
		fields        fields
		wantWarnings  []error
		wantErrorList []error
	}{
		{
			name: "invalid host",
			fields: fields{
				apihost: "1.2.3.4:5678",
			},
			wantWarnings:  nil,
			wantErrorList: []error{errors.New("ConfigMap/cluster-info.data.kubeconfig.clusters[0].cluster.server field in namespace kube-public should start with http:// or https://, please edit it first")},
		},
		{
			name: "valid host",
			fields: fields{
				apihost: "https://1.2.3.4:5678",
			},
			wantWarnings:  nil,
			wantErrorList: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := KlusterletApiserverCheck{
				KlusterletApiserver: tt.fields.apihost,
			}
			gotWarnings, gotErrorList := c.Check()
			testinghelper.AssertErrors(t, gotWarnings, tt.wantWarnings)
			testinghelper.AssertErrors(t, gotErrorList, tt.wantErrorList)
		})
	}
}
