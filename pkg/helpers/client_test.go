// Copyright Contributors to the Open Cluster Management project

package helpers

import (
	"errors"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func Test_WatchUntil(t *testing.T) {
	isAdded := func(event watch.Event) bool {
		return event.Type == watch.Added
	}

	t.Run("returns nil once the condition is satisfied", func(t *testing.T) {
		w := watch.NewFake()
		go func() {
			w.Modify(&corev1.Pod{})
			w.Add(&corev1.Pod{})
		}()

		err := WatchUntil(func() (watch.Interface, error) { return w, nil }, isAdded)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("ignores non matching events until the channel closes", func(t *testing.T) {
		w := watch.NewFake()
		go func() {
			w.Modify(&corev1.Pod{})
			w.Modify(&corev1.Pod{})
			w.Stop()
		}()

		err := WatchUntil(func() (watch.Interface, error) { return w, nil }, isAdded)
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if !strings.Contains(err.Error(), "closed") {
			t.Fatalf("expected the error to describe a closed channel, got %q", err.Error())
		}
	})

	t.Run("returns the watchFunc error without starting a watch", func(t *testing.T) {
		wantErr := errors.New("watch creation failed")
		err := WatchUntil(func() (watch.Interface, error) { return nil, wantErr }, isAdded)
		if err != wantErr {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}
