// Copyright Contributors to the Open Cluster Management project

package helpers

import (
	"fmt"
	"os"
)

func GetExampleHeader() string {
	switch os.Args[0] {
	case "oc":
		return "oc cm"
	case "kubectl":
		return "kubectl cm"
	default:
		return os.Args[0]
	}
}

func DryRunMessage(dryRun bool) {
	if dryRun {
		fmt.Printf("%s is running in dry-run mode\n", GetExampleHeader())
	}
}
