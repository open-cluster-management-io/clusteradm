package common

import (
	"strings"

	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

const (
	// SelfManagedClusterLabelKey is the label key to identify self-managed clusters
	SelfManagedClusterLabelKey = "local-cluster"
)

// IsClusterSelfManaged checks if the given ManagedCluster is a self-managed (local) cluster
// by checking if it has the "local-cluster" label set to "true"
func IsClusterSelfManaged(cluster *clusterv1.ManagedCluster) bool {
	if len(cluster.Labels) == 0 {
		return false
	}
	val, ok := cluster.Labels[SelfManagedClusterLabelKey]
	return ok && strings.EqualFold(val, "true")
}
