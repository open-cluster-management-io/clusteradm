// Copyright Contributors to the Open Cluster Management project
package printer

import (
	"bytes"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
)

func TestPrefixWriterWrite(t *testing.T) {
	cases := []struct {
		name  string
		level int
		want  string
	}{
		{name: "level 0 has no indent", level: LEVEL_0, want: "hello\n"},
		{name: "level 1 has one indent", level: LEVEL_1, want: "  hello\n"},
		{name: "level 2 has two indents", level: LEVEL_2, want: "    hello\n"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			pw := NewPrefixWriter(buf)
			pw.Write(c.level, "hello\n")
			if buf.String() != c.want {
				t.Fatalf("got %q, want %q", buf.String(), c.want)
			}
		})
	}
}

func TestPrefixWriterWriteLine(t *testing.T) {
	buf := &bytes.Buffer{}
	pw := NewPrefixWriter(buf)
	pw.WriteLine("a", "b")
	if got, want := buf.String(), "a b\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

type flushRecorder struct {
	bytes.Buffer
	flushed bool
}

func (f *flushRecorder) Flush() {
	f.flushed = true
}

func TestPrefixWriterFlush(t *testing.T) {
	rec := &flushRecorder{}
	pw := NewPrefixWriter(rec)
	pw.Flush()
	if !rec.flushed {
		t.Fatalf("expected Flush to be called on an out that implements flusher")
	}
}

func TestPrefixWriterFlushWithoutFlusher(t *testing.T) {
	buf := &bytes.Buffer{}
	pw := NewPrefixWriter(buf)
	pw.Flush()
}

func TestGetSpinnerPodStatus(t *testing.T) {
	cases := []struct {
		name string
		pod  *corev1.Pod
		want string
	}{
		{
			name: "phase with no waiting container",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{Phase: corev1.PodRunning},
			},
			want: "Running",
		},
		{
			name: "waiting container reason overrides phase",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}}},
					},
				},
			},
			want: "ImagePullBackOff",
		},
		{
			name: "last waiting container wins",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "First"}}},
						{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "Second"}}},
					},
				},
			},
			want: "Second",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GetSpinnerPodStatus(c.pod); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestGetSpinnerKlusterletStatus(t *testing.T) {
	cond := func(t, reason string) metav1.Condition {
		return metav1.Condition{Type: t, Status: metav1.ConditionFalse, Reason: reason}
	}

	cases := []struct {
		name string
		kl   *operatorv1.Klusterlet
		want string
	}{
		{
			name: "no conditions",
			kl:   &operatorv1.Klusterlet{},
			want: "",
		},
		{
			name: "registration desired degraded takes priority",
			kl: &operatorv1.Klusterlet{Status: operatorv1.KlusterletStatus{Conditions: []metav1.Condition{
				cond("RegistrationDesiredDegraded", "reg-degraded"),
				cond("WorkDesiredDegraded", "work-degraded"),
				cond("Available", "avail"),
				cond("Applied", "applied"),
			}}},
			want: "reg-degraded",
		},
		{
			name: "work desired degraded when registration missing",
			kl: &operatorv1.Klusterlet{Status: operatorv1.KlusterletStatus{Conditions: []metav1.Condition{
				cond("WorkDesiredDegraded", "work-degraded"),
				cond("Available", "avail"),
			}}},
			want: "work-degraded",
		},
		{
			name: "available when only available and applied present",
			kl: &operatorv1.Klusterlet{Status: operatorv1.KlusterletStatus{Conditions: []metav1.Condition{
				cond("Available", "avail"),
				cond("Applied", "applied"),
			}}},
			want: "avail",
		},
		{
			name: "applied as last resort",
			kl: &operatorv1.Klusterlet{Status: operatorv1.KlusterletStatus{Conditions: []metav1.Condition{
				cond("Applied", "applied"),
			}}},
			want: "applied",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GetSpinnerKlusterletStatus(c.kl); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}
