package util

import (
	proxyv1alpha1 "open-cluster-management.io/cluster-proxy/pkg/apis/proxy/v1alpha1"
)

func IsServiceResolverLegal(mpsr *proxyv1alpha1.ManagedProxyServiceResolver) bool {
	// Check managed cluster selector
	if mpsr.Spec.ManagedClusterSelector.Type != proxyv1alpha1.ManagedClusterSelectorTypeClusterSet {
		return false
	}
	if mpsr.Spec.ManagedClusterSelector.ManagedClusterSet == nil {
		return false
	}
	// Check service selector
	if mpsr.Spec.ServiceSelector.Type != proxyv1alpha1.ServiceSelectorTypeServiceRef {
		return false
	}
	if mpsr.Spec.ServiceSelector.ServiceRef == nil {
		return false
	}
	return true
}
