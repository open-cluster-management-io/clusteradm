package util

import (
	"crypto/sha256"
	"fmt"
)

func GenerateServiceURL(cluster, namespace, service string) string {
	// Using hash to generate a random string;
	// Sum256 will give a string with length equals 64. But the name of a service must be no more than 63 characters.
	// Also need to add "cluster-proxy-" as prefix to prevent content starts with a number.
	content := sha256.Sum256([]byte(fmt.Sprintf("%s %s %s", cluster, namespace, service)))
	return fmt.Sprintf("cluster-proxy-%x", content)[:63]
}

// Using SHA256 to hash cluster.name to:
// 1. Generate consistent and unique host names
// 2. Keep host name length under DNS limit (max 64 chars)
func GenerateServiceProxyHost(clusterName string) string {
	return fmt.Sprintf("cluster-%x",
		sha256.Sum256([]byte(clusterName)))[:64-len("cluster-")] + ".open-cluster-management.proxy"
}
