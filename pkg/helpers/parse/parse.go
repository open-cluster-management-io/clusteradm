// Copyright Contributors to the Open Cluster Management project
package parse

import (
	"fmt"
	"strings"
)

//nolint:revive
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

// ParseUniqueNames parses a comma-separated list
// of names and returns a slice of unique non-empty names.
func ParseUniqueNames(names []string) []string {
	sawName := map[string]bool{}
	nameList := []string{}
	for _, n := range names {
		name := strings.TrimSpace(n)
		if name == "" || sawName[name] {
			continue
		}
		sawName[name] = true
		nameList = append(nameList, name)
	}
	return nameList
}
