// Copyright Contributors to the Open Cluster Management project
package check

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"

	clusterfake "open-cluster-management.io/api/client/cluster/clientset/versioned/fake"
	operatorfake "open-cluster-management.io/api/client/operator/clientset/versioned/fake"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
)

func newClusterFakeWithResources(resources []*metav1.APIResourceList) *clusterfake.Clientset {
	client := clusterfake.NewSimpleClientset()
	client.Discovery().(*fakediscovery.FakeDiscovery).Resources = resources
	return client
}

func newOperatorFakeWithResources(resources []*metav1.APIResourceList) *operatorfake.Clientset {
	client := operatorfake.NewSimpleClientset()
	client.Discovery().(*fakediscovery.FakeDiscovery).Resources = resources
	return client
}

func TestCheckForHub(t *testing.T) {
	cases := []struct {
		name      string
		resources []*metav1.APIResourceList
		wantErr   bool
	}{
		{
			name: "managedclusters resource present",
			resources: []*metav1.APIResourceList{
				{
					GroupVersion: clusterv1.GroupVersion.String(),
					APIResources: []metav1.APIResource{{Name: ManagedClusterResourceName}},
				},
			},
			wantErr: false,
		},
		{
			name: "group version present but resource missing",
			resources: []*metav1.APIResourceList{
				{
					GroupVersion: clusterv1.GroupVersion.String(),
					APIResources: []metav1.APIResource{{Name: "somethingelse"}},
				},
			},
			wantErr: true,
		},
		{
			name:      "group version not found on non-hub cluster",
			resources: nil,
			wantErr:   true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := newClusterFakeWithResources(c.resources)
			err := CheckForHub(client)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestCheckForManagedCluster(t *testing.T) {
	cases := []struct {
		name      string
		resources []*metav1.APIResourceList
		wantErr   bool
	}{
		{
			name: "clusterclaims resource present",
			resources: []*metav1.APIResourceList{
				{
					GroupVersion: clusterv1alpha1.GroupVersion.String(),
					APIResources: []metav1.APIResource{{Name: ClusterClaimResourceName}},
				},
			},
			wantErr: false,
		},
		{
			name: "group version present but resource missing",
			resources: []*metav1.APIResourceList{
				{
					GroupVersion: clusterv1alpha1.GroupVersion.String(),
					APIResources: []metav1.APIResource{{Name: "somethingelse"}},
				},
			},
			wantErr: true,
		},
		{
			name:      "group version not found on non-managed cluster",
			resources: nil,
			wantErr:   true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := newClusterFakeWithResources(c.resources)
			err := CheckForManagedCluster(client)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestCheckForKlusterletCRD(t *testing.T) {
	cases := []struct {
		name      string
		resources []*metav1.APIResourceList
		wantErr   bool
	}{
		{
			name: "klusterlets resource present",
			resources: []*metav1.APIResourceList{
				{
					GroupVersion: operatorv1.GroupVersion.String(),
					APIResources: []metav1.APIResource{{Name: KlusterletResourceName}},
				},
			},
			wantErr: false,
		},
		{
			name: "group version present but resource missing",
			resources: []*metav1.APIResourceList{
				{
					GroupVersion: operatorv1.GroupVersion.String(),
					APIResources: []metav1.APIResource{{Name: "somethingelse"}},
				},
			},
			wantErr: true,
		},
		{
			name:      "group version not found",
			resources: nil,
			wantErr:   true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := newOperatorFakeWithResources(c.resources)
			err := CheckForKlusterletCRD(client)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestIsFeatureEnabled(t *testing.T) {
	cases := []struct {
		name         string
		featureGates []operatorv1.FeatureGate
		feature      string
		want         bool
	}{
		{
			name:         "empty feature gates",
			featureGates: nil,
			feature:      "ManifestWorkReplicaSet",
			want:         false,
		},
		{
			name: "feature enabled",
			featureGates: []operatorv1.FeatureGate{
				{Feature: "ManifestWorkReplicaSet", Mode: operatorv1.FeatureGateModeTypeEnable},
			},
			feature: "ManifestWorkReplicaSet",
			want:    true,
		},
		{
			name: "feature explicitly disabled",
			featureGates: []operatorv1.FeatureGate{
				{Feature: "AddonManagement", Mode: operatorv1.FeatureGateModeTypeDisable},
			},
			feature: "AddonManagement",
			want:    false,
		},
		{
			name: "feature not present among other gates",
			featureGates: []operatorv1.FeatureGate{
				{Feature: "SomeOtherFeature", Mode: operatorv1.FeatureGateModeTypeEnable},
			},
			feature: "AddonManagement",
			want:    false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := IsFeatureEnabled(c.featureGates, c.feature)
			if got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}
}
