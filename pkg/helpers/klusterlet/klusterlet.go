// Copyright Contributors to the Open Cluster Management project
package klusterlet

import (
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"k8s.io/klog/v2"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

// MergeKlusterletFile reads a Klusterlet YAML file and merges its spec into the provided klusterletChartConfig.
func MergeKlusterletFile(klusterletFile string, klusterletChartConfig *chart.KlusterletChartConfig) error {
	yamlBytes, err := os.ReadFile(klusterletFile)
	if err != nil {
		return fmt.Errorf("failed to read klusterlet file %s: %v", klusterletFile, err)
	}
	var klusterlet operatorv1.Klusterlet
	if err := yaml.Unmarshal(yamlBytes, &klusterlet); err != nil {
		return fmt.Errorf("failed to parse klusterlet YAML: %v", err)
	}

	// merge the spec fields into the chart config
	spec := klusterlet.Spec

	if spec.Namespace != "" {
		klusterletChartConfig.Klusterlet.Namespace = spec.Namespace
	}
	if spec.ClusterName != "" {
		klusterletChartConfig.Klusterlet.ClusterName = spec.ClusterName
	}
	if spec.DeployOption.Mode != "" {
		klusterletChartConfig.Klusterlet.Mode = spec.DeployOption.Mode
	}
	if len(spec.ExternalServerURLs) > 0 {
		klusterletChartConfig.Klusterlet.ExternalServerURLs = spec.ExternalServerURLs
	}
	if spec.NodePlacement.NodeSelector != nil || spec.NodePlacement.Tolerations != nil {
		klusterletChartConfig.Klusterlet.NodePlacement = spec.NodePlacement
	}
	if spec.RegistrationConfiguration != nil {
		klusterletChartConfig.Klusterlet.RegistrationConfiguration = *spec.RegistrationConfiguration
	}
	if spec.WorkConfiguration != nil {
		klusterletChartConfig.Klusterlet.WorkConfiguration = *spec.WorkConfiguration
	}
	if spec.ResourceRequirement != nil {
		klusterletChartConfig.Klusterlet.ResourceRequirement = spec.ResourceRequirement
	}
	if spec.PriorityClassName != "" {
		klusterletChartConfig.PriorityClassName = spec.PriorityClassName
	}

	klog.V(2).InfoS("Successfully merged klusterlet file into klusterlet chart config", "file", klusterletFile)
	return nil
}
