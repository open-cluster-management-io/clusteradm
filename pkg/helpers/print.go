// Copyright Contributors to the Open Cluster Management project
package helpers

import (
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	corev1 "k8s.io/api/core/v1"
)

var suffixColor = color.New(color.Bold, color.FgGreen)

func NewSpinner(suffix string, interval time.Duration) *spinner.Spinner {
	return spinner.New(
		spinner.CharSets[14],
		interval,
		spinner.WithColor("green"),
		spinner.WithHiddenCursor(true),
		spinner.WithSuffix(suffixColor.Sprintf(" %s", suffix)))
}

func NewSpinnerWithStatus(suffix string, interval time.Duration, final string, statusFunc func() string) *spinner.Spinner {
	s := NewSpinner(suffix, interval)
	s.FinalMSG = final
	s.PreUpdate = func(s *spinner.Spinner) {
		status := statusFunc()
		if len(status) > 0 {
			s.Suffix = suffixColor.Sprintf(" %s (%s)", suffix, status)
		} else {
			s.Suffix = suffixColor.Sprintf(" %s", suffix)
		}
	}
	return s
}

func GetSpinnerPodStatus(pod *corev1.Pod) string {
	reason := string(pod.Status.Phase)
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting != nil {
			reason = containerStatus.State.Waiting.Reason
		}
	}
	return reason
}
