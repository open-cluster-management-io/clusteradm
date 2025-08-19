// Copyright Contributors to the Open Cluster Management project
package parse

import (
	"fmt"
	"strings"
)

func ParseLabels(labels []string) (map[string]string, error) {
	labelMap := make(map[string]string)
	for _, labelString := range labels {
		labelSlice := strings.Split(labelString, "=")
		if len(labelSlice) != 2 {
			return nil, fmt.Errorf("error parsing label '%s'. Expected to be of the form: key=value", labelString)
		}
		labelMap[labelSlice[0]] = labelSlice[1]
	}
	return labelMap, nil
}
