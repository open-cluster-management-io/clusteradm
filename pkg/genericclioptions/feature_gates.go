// Copyright Contributors to the Open Cluster Management project
package genericclioptions

import (
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/component-base/featuregate"
	ocmfeature "open-cluster-management.io/api/feature"
	operatorv1 "open-cluster-management.io/api/operator/v1"
)

var HubMutableFeatureGate = featuregate.NewFeatureGate()
var SpokeMutableFeatureGate = featuregate.NewFeatureGate()

func init() {
	utilruntime.Must(HubMutableFeatureGate.Add(ocmfeature.DefaultHubWorkFeatureGates))
	utilruntime.Must(HubMutableFeatureGate.Add(ocmfeature.DefaultHubRegistrationFeatureGates))
	utilruntime.Must(SpokeMutableFeatureGate.Add(ocmfeature.DefaultSpokeRegistrationFeatureGates))
	utilruntime.Must(SpokeMutableFeatureGate.Add(ocmfeature.DefaultHubWorkFeatureGates))

	// Update default features
	utilruntime.Must(HubMutableFeatureGate.SetFromMap(map[string]bool{string(ocmfeature.DefaultClusterSet): true}))
	utilruntime.Must(SpokeMutableFeatureGate.SetFromMap(map[string]bool{string(ocmfeature.AddonManagement): true}))
}

func ConvertToFeatureGateAPI(featureGates featuregate.MutableFeatureGate, defaultFeatureGate map[featuregate.Feature]featuregate.FeatureSpec) []operatorv1.FeatureGate {
	var features []operatorv1.FeatureGate
	for feature := range featureGates.GetAll() {
		spec, ok := defaultFeatureGate[feature]
		if !ok {
			continue
		}

		if featureGates.Enabled(feature) && !spec.Default {
			features = append(features, operatorv1.FeatureGate{Feature: string(feature), Mode: operatorv1.FeatureGateModeTypeEnable})
		} else if !featureGates.Enabled(feature) && spec.Default {
			features = append(features, operatorv1.FeatureGate{Feature: string(feature), Mode: operatorv1.FeatureGateModeTypeDisable})
		}
	}

	return features
}
