// Copyright Contributors to the Open Cluster Management project
package check

import (
	"errors"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/component-base/featuregate"

	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
)

func newFakeDiscovery(resources []*metav1.APIResourceList, err error) discovery.DiscoveryInterface {
	fake := &k8stesting.Fake{Resources: resources}
	if err != nil {
		fake.PrependReactor("get", "resource", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, err
		})
	}
	return &fakediscovery.FakeDiscovery{Fake: fake}
}

func TestCheckForHub(t *testing.T) {
	cases := []struct {
		name        string
		resources   []*metav1.APIResourceList
		discoverErr error
		wantErr     bool
		wantErrText string
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
		{
			name:        "non not-found discovery error is wrapped with the group version",
			discoverErr: errors.New("connection refused"),
			wantErr:     true,
			wantErrText: clusterv1.GroupVersion.String(),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := newFakeDiscovery(c.resources, c.discoverErr)
			err := CheckForHub(client)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if c.wantErrText != "" && !strings.Contains(err.Error(), c.wantErrText) {
				t.Fatalf("expected error to contain %q, got %q", c.wantErrText, err.Error())
			}
		})
	}
}

func TestCheckForManagedCluster(t *testing.T) {
	cases := []struct {
		name           string
		resources      []*metav1.APIResourceList
		discoverErr    error
		wantErr        bool
		wantErrNotText string
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
		{
			name:           "non not-found discovery error drops the original error text",
			discoverErr:    errors.New("connection refused"),
			wantErr:        true,
			wantErrNotText: "connection refused",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := newFakeDiscovery(c.resources, c.discoverErr)
			err := CheckForManagedCluster(client)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if c.wantErrNotText != "" && strings.Contains(err.Error(), c.wantErrNotText) {
				t.Fatalf("expected error not to contain %q, got %q", c.wantErrNotText, err.Error())
			}
		})
	}
}

func TestCheckForKlusterletCRD(t *testing.T) {
	cases := []struct {
		name        string
		resources   []*metav1.APIResourceList
		discoverErr error
		wantErr     bool
		wantErrIs   error
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
		{
			name:        "non not-found discovery error is returned unchanged",
			discoverErr: errors.New("connection refused"),
			wantErr:     true,
			wantErrIs:   errors.New("connection refused"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := newFakeDiscovery(c.resources, c.discoverErr)
			err := CheckForKlusterletCRD(client)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if c.discoverErr != nil && !errors.Is(err, c.discoverErr) {
				t.Fatalf("expected the original discovery error to be returned unchanged, got %v", err)
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

func TestIsFeatureEnabledWithDefault(t *testing.T) {
	defaults := map[featuregate.Feature]featuregate.FeatureSpec{
		"DefaultOn":  {Default: true},
		"DefaultOff": {Default: false},
	}
	cases := []struct {
		name         string
		featureGates []operatorv1.FeatureGate
		feature      featuregate.Feature
		want         bool
	}{
		{
			name:    "unset feature that is on by default",
			feature: "DefaultOn",
			want:    true,
		},
		{
			name:    "unset feature that is off by default",
			feature: "DefaultOff",
			want:    false,
		},
		{
			name: "feature on by default but explicitly disabled",
			featureGates: []operatorv1.FeatureGate{
				{Feature: "DefaultOn", Mode: operatorv1.FeatureGateModeTypeDisable},
			},
			feature: "DefaultOn",
			want:    false,
		},
		{
			name: "feature off by default but explicitly enabled",
			featureGates: []operatorv1.FeatureGate{
				{Feature: "DefaultOff", Mode: operatorv1.FeatureGateModeTypeEnable},
			},
			feature: "DefaultOff",
			want:    true,
		},
		{
			name: "other gates set, feature falls back to its default",
			featureGates: []operatorv1.FeatureGate{
				{Feature: "DefaultOff", Mode: operatorv1.FeatureGateModeTypeEnable},
			},
			feature: "DefaultOn",
			want:    true,
		},
		{
			name:    "unknown feature",
			feature: "Unknown",
			want:    false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := IsFeatureEnabledWithDefault(c.featureGates, defaults, c.feature)
			if got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}
}
