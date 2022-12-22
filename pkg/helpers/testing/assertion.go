// Copyright Contributors to the Open Cluster Management project
package testing

import (
	"testing"

	clienttesting "k8s.io/client-go/testing"
)

func AssertAction(t *testing.T, actual clienttesting.Action, expected string) {
	if actual.GetVerb() != expected {
		t.Errorf("expected %s action but got: %#v", expected, actual)
	}
}

func AssertWarnings(t *testing.T, actual, expected []string) {
	if len(actual) != len(expected) {
		t.Errorf("expected %v but got: %v", expected, actual)
	}
	for i := 0; i < len(actual); i++ {
		if actual[i] != expected[i] {
			t.Errorf("expected %v but got: %v", expected, actual)
		}
	}
}

func AssertErrors(t *testing.T, actual, expected []error) {
	if len(actual) != len(expected) {
		t.Errorf("expected %v but got: %v", expected, actual)
	}
	for i := 0; i < len(actual); i++ {
		if actual[i].Error() != expected[i].Error() {
			t.Errorf("expected %v but got: %v", expected, actual)
		}
	}
}
