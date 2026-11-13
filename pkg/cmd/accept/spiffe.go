// Copyright Contributors to the Open Cluster Management project
package accept

import (
	"encoding/json"
	"fmt"

	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

// spiffeTrustDomainAnnotation pins a ManagedCluster to the trust domain its agents registered
// from. The hub sets it when a SPIFFE agent creates the cluster.
const spiffeTrustDomainAnnotation = "open-cluster-management.io/spiffe-trust-domain"

// spiffeAcceptance decides how to accept a cluster with the spiffe driver. A spiffe cluster has
// no CSR: its agents use their SVIDs. With --spiffe-trust-domain, a cluster pinned to another
// trust domain is refused, and an unpinned one gets the pin (returned as a merge patch).
func spiffeAcceptance(mc *clusterv1.ManagedCluster, trustDomain string) (isSPIFFE bool, pinPatch []byte, err error) {
	pinned, hasPin := mc.Annotations[spiffeTrustDomainAnnotation]
	if trustDomain == "" {
		return hasPin, nil, nil
	}
	if hasPin {
		if pinned != trustDomain {
			return true, nil, fmt.Errorf("managed cluster %s registered from trust domain %s, not %s; not accepted",
				mc.Name, pinned, trustDomain)
		}
		return true, nil, nil
	}
	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{"annotations": map[string]string{spiffeTrustDomainAnnotation: trustDomain}}})
	return true, patch, err
}
