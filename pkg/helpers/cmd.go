// Copyright Contributors to the Open Cluster Management project

package helpers

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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

func UsageTempate(cmd *cobra.Command, reader ScenarioReader, valuesTemplatePath string) string {
	baseUsage := cmd.UsageTemplate()
	b, err := reader.Asset(valuesTemplatePath)
	if err != nil {
		return fmt.Sprintf("%s\n\n Values template:\n%s", baseUsage, err.Error())
	}
	return fmt.Sprintf("%s\n\n Values template:\n%s", baseUsage, string(b))
}
