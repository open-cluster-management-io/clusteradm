// Copyright Contributors to the Open Cluster Management project
package helpers

import (
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

func NewSpinner(suffix string, interval time.Duration) *spinner.Spinner {
	suffixColor := color.New(color.Bold, color.FgGreen)
	return spinner.New(
		spinner.CharSets[14],
		interval,
		spinner.WithColor("green"),
		spinner.WithHiddenCursor(true),
		spinner.WithSuffix(suffixColor.Sprintf(" %s", suffix)))
}
