// Copyright Contributors to the Open Cluster Management project
package accept

import (
	"context"
	"encoding/json"
	"fmt"

	certificatesv1 "k8s.io/api/certificates/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

// spiffeTrustDomainAnnotation pins a ManagedCluster to the trust domain its agents registered
// from. The hub sets it when a SPIFFE agent creates the cluster.
const spiffeTrustDomainAnnotation = "open-cluster-management.io/spiffe-trust-domain"

// spiffeAcceptance decides how to accept a cluster with the spiffe driver. A spiffe cluster has
// no CSR: its agents use their SVIDs. With --spiffe-trust-domain, a cluster pinned to another
// trust domain is refused, and an unpinned one gets the pin (returned as a merge patch that
// carries the resourceVersion read, so it fails if the cluster changed meanwhile). An unpinned
// cluster with a pending CSR registered with csr: it isn't pinned, and its CSR is approved as before.
func spiffeAcceptance(mc *clusterv1.ManagedCluster, trustDomain string, pendingCSR bool) (isSPIFFE bool, pinPatch []byte, err error) {
	pinned, hasPin := mc.Annotations[spiffeTrustDomainAnnotation]
	if trustDomain == "" || (!hasPin && pendingCSR) {
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
		"metadata": map[string]any{
			"resourceVersion": mc.ResourceVersion,
			"annotations":     map[string]string{spiffeTrustDomainAnnotation: trustDomain},
		}})
	return true, patch, err
}

// hasPendingCSR reports whether the cluster has a CSR that is neither approved nor denied: it is
// registering with the csr driver.
func hasPendingCSR(ctx context.Context, kubeClient kubernetes.Interface, clusterName string) (bool, error) {
	csrs, err := kubeClient.CertificatesV1().CertificateSigningRequests().List(ctx,
		metav1.ListOptions{LabelSelector: fmt.Sprintf("%v = %v", clusterLabel, clusterName)})
	if err != nil {
		return false, fmt.Errorf("fail to list the csrs of cluster %s: %w", clusterName, err)
	}
	for _, csr := range csrs.Items {
		decided := false
		for _, c := range csr.Status.Conditions {
			if c.Type == certificatesv1.CertificateApproved || c.Type == certificatesv1.CertificateDenied {
				decided = true
			}
		}
		if !decided {
			return true, nil
		}
	}
	return false, nil
}
