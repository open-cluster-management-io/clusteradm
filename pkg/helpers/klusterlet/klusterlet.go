// Copyright Contributors to the Open Cluster Management project
package klusterlet

import (
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"k8s.io/klog/v2"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

// MergeKlusterletValues merges a klusterlet values file into the klusterlet chart config.
func MergeKlusterletValues(klusterletValuesFile string, klusterletChartConfig *chart.KlusterletChartConfig) error {
	values, err := os.ReadFile(klusterletValuesFile)
	if err != nil {
		return fmt.Errorf("failed to read klusterlet values file %s: %v", klusterletValuesFile, err)
	}
	if err := yaml.Unmarshal(values, klusterletChartConfig); err != nil {
		return fmt.Errorf("failed to unmarshal klusterlet values: %v", err)
	}
	klog.V(2).InfoS("Successfully merged klusterlet values file into klusterlet chart config", "file", klusterletValuesFile)
	return nil
}
