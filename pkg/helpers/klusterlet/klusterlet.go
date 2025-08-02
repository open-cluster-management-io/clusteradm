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

// MergeKlusterletFileJoin merges the klusterlet file into the klusterlet chart config for a klusterlet join.
// All fields are allowed to be overridden during a klusterlet join.
func MergeKlusterletFileJoin(klusterletFile string, klusterletChartConfig *chart.KlusterletChartConfig) error {
	spec, err := loadKlusterletFile(klusterletFile)
	if err != nil {
		return fmt.Errorf("failed to load klusterlet file %s: %v", klusterletFile, err)
	}

	if err := mergeKlusterletFileUpgrade(spec, klusterletChartConfig); err != nil {
		return fmt.Errorf("failed to merge klusterlet file %s: %v", klusterletFile, err)
	}
	if err := mergeKlusterletFileJoin(spec, klusterletChartConfig); err != nil {
		return fmt.Errorf("failed to merge klusterlet file %s: %v", klusterletFile, err)
	}

	klog.V(2).InfoS("Successfully merged klusterlet file into klusterlet chart config for join", "file", klusterletFile)
	return nil
}

// MergeKlusterletFileUpgrade merges the klusterlet file into the klusterlet chart config for a klusterlet upgrade.
// Overriding the cluster name, klusterlet namespace, and priority class name is not allowed during upgrades.
func MergeKlusterletFileUpgrade(klusterletFile string, klusterletChartConfig *chart.KlusterletChartConfig) error {
	spec, err := loadKlusterletFile(klusterletFile)
	if err != nil {
		return fmt.Errorf("failed to load klusterlet file %s: %v", klusterletFile, err)
	}

	if err := mergeKlusterletFileUpgrade(spec, klusterletChartConfig); err != nil {
		return fmt.Errorf("failed to merge klusterlet file %s: %v", klusterletFile, err)
	}

	klog.V(2).InfoS("Successfully merged klusterlet file into klusterlet chart config for upgrade", "file", klusterletFile)
	return nil
}

func loadKlusterletFile(klusterletFile string) (*operatorv1.KlusterletSpec, error) {
	yamlBytes, err := os.ReadFile(klusterletFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read klusterlet file %s: %v", klusterletFile, err)
	}
	var klusterlet operatorv1.Klusterlet
	if err := yaml.Unmarshal(yamlBytes, &klusterlet); err != nil {
		return nil, fmt.Errorf("failed to parse klusterlet YAML: %v", err)
	}
	return &klusterlet.Spec, nil
}

func mergeKlusterletFileJoin(spec *operatorv1.KlusterletSpec, klusterletChartConfig *chart.KlusterletChartConfig) error {
	if spec.Namespace != "" {
		klusterletChartConfig.Klusterlet.Namespace = spec.Namespace
	}
	if spec.ClusterName != "" {
		klusterletChartConfig.Klusterlet.ClusterName = spec.ClusterName
	}
	if spec.PriorityClassName != "" {
		klusterletChartConfig.PriorityClassName = spec.PriorityClassName
	}
	return nil
}

func mergeKlusterletFileUpgrade(spec *operatorv1.KlusterletSpec, klusterletChartConfig *chart.KlusterletChartConfig) error {
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
		if spec.RegistrationConfiguration.ClusterClaimConfiguration != nil {
			if spec.RegistrationConfiguration.ClusterClaimConfiguration.MaxCustomClusterClaims == 0 {
				klusterletChartConfig.Klusterlet.RegistrationConfiguration.ClusterClaimConfiguration.MaxCustomClusterClaims = 20
			}
		}
		if spec.RegistrationConfiguration.RegistrationDriver.AuthType == "" {
			klusterletChartConfig.Klusterlet.RegistrationConfiguration.RegistrationDriver.AuthType = "csr"
		}
		if spec.RegistrationConfiguration.BootstrapKubeConfigs.Type == "" {
			klusterletChartConfig.Klusterlet.RegistrationConfiguration.BootstrapKubeConfigs.Type = operatorv1.None
		}
	}
	if spec.WorkConfiguration != nil {
		klusterletChartConfig.Klusterlet.WorkConfiguration = *spec.WorkConfiguration
	}
	if spec.ResourceRequirement != nil {
		klusterletChartConfig.Klusterlet.ResourceRequirement = spec.ResourceRequirement
		if spec.ResourceRequirement.Type == "" {
			klusterletChartConfig.Klusterlet.ResourceRequirement.Type = operatorv1.ResourceQosClassDefault
		}
	}
	return nil
}
