// Copyright Contributors to the Open Cluster Management project
package klusterlet

import (
	"os"
	"path/filepath"
	"testing"

	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

func TestMergeKlusterletFile(t *testing.T) {
	// Create a temporary klusterlet YAML file
	tmpDir := t.TempDir()
	klusterletFile := filepath.Join(tmpDir, "test-klusterlet.yaml")

	klusterletYAML := `apiVersion: operator.open-cluster-management.io/v1
kind: Klusterlet
metadata:
  name: test-klusterlet
spec:
  clusterName: test-cluster
  namespace: test-namespace
  deployOption:
    mode: Singleton
  registrationConfiguration:
    clientCertExpirationSeconds: 86400
    featureGates:
    - feature: DefaultClusterSet
      mode: Enable
  workConfiguration:
    featureGates:
    - feature: ManifestWorkReplicaSet
      mode: Enable
    statusSyncInterval: 60s
  resourceRequirement:
    type: ResourceRequirement
    resourceRequirements:
      requests:
        cpu: 100m
        memory: 128Mi
  priorityClassName: system-cluster-critical`

	if err := os.WriteFile(klusterletFile, []byte(klusterletYAML), 0644); err != nil {
		t.Fatalf("Failed to write test klusterlet file: %v", err)
	}

	// Create a default chart config
	chartConfig := chart.NewDefaultKlusterletChartConfig()

	// Test the merge function
	err := MergeKlusterletFileJoin(klusterletFile, chartConfig)
	if err != nil {
		t.Fatalf("MergeKlusterletFileJoin failed: %v", err)
	}

	// Verify the merge results
	if chartConfig.Klusterlet.ClusterName != "test-cluster" {
		t.Errorf("Expected ClusterName to be 'test-cluster', got '%s'", chartConfig.Klusterlet.ClusterName)
	}

	if chartConfig.Klusterlet.Namespace != "test-namespace" {
		t.Errorf("Expected Namespace to be 'test-namespace', got '%s'", chartConfig.Klusterlet.Namespace)
	}

	if chartConfig.Klusterlet.Mode != operatorv1.InstallModeSingleton {
		t.Errorf("Expected Mode to be 'Singleton', got '%s'", chartConfig.Klusterlet.Mode)
	}

	if chartConfig.Klusterlet.RegistrationConfiguration.ClientCertExpirationSeconds != 86400 {
		t.Errorf("Expected ClientCertExpirationSeconds to be 86400, got %d", chartConfig.Klusterlet.RegistrationConfiguration.ClientCertExpirationSeconds)
	}

	if len(chartConfig.Klusterlet.RegistrationConfiguration.FeatureGates) != 1 {
		t.Errorf("Expected 1 registration feature gate, got %d", len(chartConfig.Klusterlet.RegistrationConfiguration.FeatureGates))
	}

	if len(chartConfig.Klusterlet.WorkConfiguration.FeatureGates) != 1 {
		t.Errorf("Expected 1 work feature gate, got %d", len(chartConfig.Klusterlet.WorkConfiguration.FeatureGates))
	}

	if chartConfig.Klusterlet.WorkConfiguration.StatusSyncInterval != nil {
		expectedInterval := "&Duration{Duration:1m0s,}"
		if chartConfig.Klusterlet.WorkConfiguration.StatusSyncInterval.String() != expectedInterval {
			t.Errorf("Expected StatusSyncInterval to be '%s', got '%s'", expectedInterval, chartConfig.Klusterlet.WorkConfiguration.StatusSyncInterval.String())
		}
	}

	if chartConfig.PriorityClassName != "system-cluster-critical" {
		t.Errorf("Expected PriorityClassName to be 'system-cluster-critical', got '%s'", chartConfig.PriorityClassName)
	}
}

func TestMergeKlusterletFileNotFound(t *testing.T) {
	chartConfig := chart.NewDefaultKlusterletChartConfig()

	err := MergeKlusterletFileJoin("/nonexistent/file.yaml", chartConfig)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestMergeKlusterletFileInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	klusterletFile := filepath.Join(tmpDir, "invalid-klusterlet.yaml")

	invalidYAML := `invalid: yaml: content: [unclosed`
	if err := os.WriteFile(klusterletFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write invalid klusterlet file: %v", err)
	}

	chartConfig := chart.NewDefaultKlusterletChartConfig()

	err := MergeKlusterletFileJoin(klusterletFile, chartConfig)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}
