// Copyright Contributors to the Open Cluster Management project
package klusterlet

import (
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"k8s.io/klog/v2"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

// MergeKlusterletValues merges a values file for the klusterlet Helm chart into a klusterlet chart config.
//
// The klusterlet chart config is a combination of default chart values and values set by clusteradm flags,
// e.g., --cluster-name, --resource-limits, --mode, etc.
//
// The values file can contain any number of valid (or invalid) klusterlet chart values. It does not necessarily
// include all values. Invalid values are ignored. Valid values override both the default chart values and the values
// set by flags.
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
