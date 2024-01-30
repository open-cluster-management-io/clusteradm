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
