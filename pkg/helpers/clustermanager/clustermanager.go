// Copyright Contributors to the Open Cluster Management project
package clustermanager

import (
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"k8s.io/klog/v2"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

// MergeClusterManagerValues merges a values file for the cluster-manager Helm chart into a cluster-manager chart config.
//
// The chart config is a combination of default chart values and values set by clusteradm flags,
// e.g., --image-registry, --resource-limits, --registration-drivers, etc.
//
// The values file can contain any subset of cluster-manager chart values. Invalid values are ignored.
// Valid values override both the default chart values and the values set by flags.
func MergeClusterManagerValues(clusterManagerValuesFile string, clusterManagerChartConfig *chart.ClusterManagerChartConfig) error {
	values, err := os.ReadFile(clusterManagerValuesFile)
	if err != nil {
		return fmt.Errorf("failed to read cluster-manager values file %s: %v", clusterManagerValuesFile, err)
	}
	if err := yaml.Unmarshal(values, clusterManagerChartConfig); err != nil {
		return fmt.Errorf("failed to unmarshal cluster-manager values: %v", err)
	}
	klog.V(2).InfoS("Successfully merged cluster-manager values file into cluster-manager chart config", "file", clusterManagerValuesFile)
	return nil
}
