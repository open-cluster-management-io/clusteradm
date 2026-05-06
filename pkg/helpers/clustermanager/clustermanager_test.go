// Copyright Contributors to the Open Cluster Management project
package clustermanager

import (
	"os"
	"path/filepath"
	"testing"

	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

func TestMergeClusterManagerValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prepare func(t *testing.T) string
		wantErr bool
		check   func(t *testing.T, cfg *chart.ClusterManagerChartConfig)
	}{
		{
			name: "valid yaml merges into config",
			prepare: func(t *testing.T) string {
				t.Helper()
				p := filepath.Join(t.TempDir(), "cluster-manager-values.yaml")
				content := "replicaCount: 2\nenableSyncLabels: true\n"
				if err := os.WriteFile(p, []byte(content), 0644); err != nil {
					t.Fatalf("write values file: %v", err)
				}
				return p
			},
			check: func(t *testing.T, cfg *chart.ClusterManagerChartConfig) {
				t.Helper()
				if cfg.ReplicaCount != 2 {
					t.Errorf("ReplicaCount: want 2, got %d", cfg.ReplicaCount)
				}
				if !cfg.EnableSyncLabels {
					t.Error("EnableSyncLabels: want true")
				}
				if !cfg.ClusterManager.Create {
					t.Error("ClusterManager.Create: want default true preserved (merge into existing config, not replace)")
				}
			},
		},
		{
			name: "missing file returns error",
			prepare: func(t *testing.T) string {
				t.Helper()
				return "/nonexistent/file.yaml"
			},
			wantErr: true,
		},
		{
			name: "invalid yaml returns error",
			prepare: func(t *testing.T) string {
				t.Helper()
				p := filepath.Join(t.TempDir(), "invalid-values.yaml")
				if err := os.WriteFile(p, []byte("invalid: yaml: content: [unclosed"), 0644); err != nil {
					t.Fatalf("write invalid file: %v", err)
				}
				return p
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := tt.prepare(t)
			cfg := chart.NewDefaultClusterManagerChartConfig()

			err := MergeClusterManagerValues(path, cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("MergeClusterManagerValues: want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("MergeClusterManagerValues: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}
