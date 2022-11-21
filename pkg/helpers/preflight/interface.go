// Copyright Contributors to the Open Cluster Management project
package preflight

import (
	"bytes"
	"fmt"
	"io"
)

// Checker validates the state of the cluster to ensure
// clusteradm will be successfully as often as possible.
type Checker interface {
	Check() (warnings []string, errorList []error)
	Name() string
}

type Error struct {
	Msg string
}

func (e Error) Error() string {
	return fmt.Sprintf("[preflight] Some fatal errors occurred:\n%s", e.Msg)
}

func (e *Error) Preflight() bool {
	return true
}

// RunChecks runs each check, display it's check/errors,
// and once all are processed will exist if any errors occured.
func RunChecks(checks []Checker, ww io.Writer) error {
	var errsBuffer bytes.Buffer
	for _, check := range checks {
		name := check.Name()
		warnings, errs := check.Check()
		for _, warning := range warnings {
			_, _ = io.WriteString(ww, fmt.Sprintf("\t[WARNING %s]: %v\n", name, warning))
		}
		for _, err := range errs {
			_, _ = errsBuffer.WriteString(fmt.Sprintf("\t[ERROR %s]: %v\n", name, err.Error()))
		}
		_, _ = io.WriteString(ww, printCheckResult(name, warnings, errs))
	}
	if errsBuffer.Len() > 0 {
		return &Error{Msg: errsBuffer.String()}
	}
	return nil
}

func printCheckResult(name string, warningList []string, errorList []error) string {
	flag := "Passed"
	warningNum, errorNum := len(warningList), len(errorList)
	if errorNum != 0 {
		flag = "Failed"
	}
	return fmt.Sprintf("Preflight check: %s %s with %d warnings and %d errors\n", name, flag, warningNum, errorNum)
}
