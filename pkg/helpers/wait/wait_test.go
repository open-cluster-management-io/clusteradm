// Copyright Contributors to the Open Cluster Management project
package wait

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"

	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
)

func TestIsPodReady(t *testing.T) {
	cases := []struct {
		name      string
		object    corev1.Pod
		wantReady bool
		wantPhase string
	}{
		{
			name: "pod without ready condition is not ready",
			object: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
				},
			},
			wantReady: false,
			wantPhase: "Pending",
		},
		{
			name: "pod with ready condition false is not ready",
			object: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{Type: corev1.PodReady, Status: corev1.ConditionFalse},
					},
				},
			},
			wantReady: false,
			wantPhase: "Running",
		},
		{
			name: "pod with ready condition true is ready",
			object: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{Type: corev1.PodInitialized, Status: corev1.ConditionTrue},
						{Type: corev1.PodReady, Status: corev1.ConditionTrue},
					},
				},
			},
			wantReady: true,
			wantPhase: "Running",
		},
		{
			name: "waiting container status overrides phase in reported status",
			object: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"},
							},
						},
					},
				},
			},
			wantReady: false,
			wantPhase: "ImagePullBackOff",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if isPodReady(c.object) != c.wantReady {
				t.Errorf("expected ready=%v, got %v", c.wantReady, isPodReady(c.object))
			}
			if got := printer.GetSpinnerPodStatus(c.object); got != c.wantPhase {
				t.Errorf("expected phase=%q, got %q", c.wantPhase, got)
			}
		})
	}
}

func TestIsFatalAPIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "not found", err: k8serrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "name"), want: false},
		{name: "forbidden", err: k8serrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "name", errors.New("no")), want: true},
		{name: "unauthorized", err: k8serrors.NewUnauthorized("no"), want: true},
		{name: "bad request", err: k8serrors.NewBadRequest("bad"), want: true},
		{name: "invalid", err: &k8serrors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonInvalid, Message: "invalid"}}, want: true},
		{name: "method not supported", err: k8serrors.NewMethodNotSupported(schema.GroupResource{Resource: "pods"}, "connect"), want: true},
		{name: "connection refused", err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}, want: false},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: false},
		{
			name: "x509 unknown authority",
			err:  &url.Error{Op: "Get", URL: "https://hub.example", Err: x509.UnknownAuthorityError{}},
			want: true,
		},
		{
			name: "x509 hostname",
			err:  x509.HostnameError{Host: "hub.example"},
			want: true,
		},
		{
			name: "tls certificate verification",
			err:  &tls.CertificateVerificationError{Err: errors.New("failed to verify certificate")},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsFatalAPIError(tt.err); got != tt.want {
				t.Fatalf("IsFatalAPIError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandlePollError(t *testing.T) {
	t.Parallel()

	retryable := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
	forbidden := k8serrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "name", errors.New("no"))

	t.Run("retryable stores status", func(t *testing.T) {
		t.Parallel()
		status := newStatus("previous")
		done, err := HandlePollError(t.Context(), retryable, status)
		if done || err != nil {
			t.Fatalf("HandlePollError() = (%v, %v), want (false, nil)", done, err)
		}
		if got := status.Load().(string); got != retryable.Error() {
			t.Fatalf("status = %q, want %q", got, retryable.Error())
		}
	})

	t.Run("forbidden aborts and stores status", func(t *testing.T) {
		t.Parallel()
		status := newStatus("previous")
		done, err := HandlePollError(t.Context(), forbidden, status)
		if done || !errors.Is(err, forbidden) {
			t.Fatalf("HandlePollError() = (%v, %v), want (false, forbidden)", done, err)
		}
		if got := status.Load().(string); got != forbidden.Error() {
			t.Fatalf("status = %q, want %q", got, forbidden.Error())
		}
	})

	t.Run("deadline on a live context is retryable", func(t *testing.T) {
		t.Parallel()
		status := newStatus("previous")
		done, err := HandlePollError(t.Context(), context.DeadlineExceeded, status)
		if done || err != nil {
			t.Fatalf("HandlePollError() = (%v, %v), want (false, nil)", done, err)
		}
		if got := status.Load().(string); got != context.DeadlineExceeded.Error() {
			t.Fatalf("status = %q, want %q", got, context.DeadlineExceeded.Error())
		}
	})

	t.Run("deadline on an expired context keeps status", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		status := newStatus("connection refused")
		done, err := HandlePollError(ctx, context.DeadlineExceeded, status)
		if done || err != nil {
			t.Fatalf("HandlePollError() = (%v, %v), want (false, nil)", done, err)
		}
		if got := status.Load().(string); got != "connection refused" {
			t.Fatalf("status = %q, want previous status", got)
		}
	})

	t.Run("nil clears status", func(t *testing.T) {
		t.Parallel()
		status := newStatus("previous")
		done, err := HandlePollError(t.Context(), nil, status)
		if done || err != nil {
			t.Fatalf("HandlePollError() = (%v, %v), want (false, nil)", done, err)
		}
		if got := status.Load().(string); got != "" {
			t.Fatalf("status = %q, want empty", got)
		}
	})
}

func TestWaitUntilPodsReady(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		pods      []runtime.Object
		wantErr   bool
		wantPhase string
	}{
		{
			name: "live ready pod succeeds while a terminating pod is ignored",
			pods: []runtime.Object{
				testPod("b-ready", corev1.PodRunning, corev1.ConditionTrue, "", false),
				testPod("a-terminating", corev1.PodFailed, corev1.ConditionTrue, "Terminating", true),
			},
			wantPhase: "Running",
		},
		{
			name: "failed pod is ignored beside a ready pod",
			pods: []runtime.Object{
				testPod("a-evicted", corev1.PodFailed, corev1.ConditionFalse, "", false),
				testPod("b-ready", corev1.PodRunning, corev1.ConditionTrue, "", false),
			},
			wantPhase: "Running",
		},
		{
			name: "succeeded pod is ignored beside a ready pod",
			pods: []runtime.Object{
				testPod("a-succeeded", corev1.PodSucceeded, corev1.ConditionFalse, "", false),
				testPod("b-ready", corev1.PodRunning, corev1.ConditionTrue, "", false),
			},
			wantPhase: "Running",
		},
		{
			name: "only failed pods",
			pods: []runtime.Object{
				testPod("old", corev1.PodFailed, corev1.ConditionFalse, "", false),
			},
			wantErr:   true,
			wantPhase: "no pods found",
		},
		{
			name: "unready pod blocks a later ready pod",
			pods: []runtime.Object{
				testPod("a-unready", corev1.PodPending, corev1.ConditionFalse, "ImagePullBackOff", false),
				testPod("b-ready", corev1.PodRunning, corev1.ConditionTrue, "", false),
			},
			wantErr:   true,
			wantPhase: "ImagePullBackOff",
		},
		{
			name: "later unready pod replaces an earlier ready status",
			pods: []runtime.Object{
				testPod("a-ready", corev1.PodRunning, corev1.ConditionTrue, "", false),
				testPod("b-unready", corev1.PodPending, corev1.ConditionFalse, "CrashLoopBackOff", false),
			},
			wantErr:   true,
			wantPhase: "CrashLoopBackOff",
		},
		{
			name:      "no pods",
			wantErr:   true,
			wantPhase: "no pods found",
		},
		{
			name: "only terminating pods",
			pods: []runtime.Object{
				testPod("old", corev1.PodRunning, corev1.ConditionTrue, "", true),
			},
			wantErr:   true,
			wantPhase: "no pods found",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			client := fake.NewSimpleClientset(c.pods...)
			ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
			defer cancel()

			phase := newStatus("")
			err := waitUntilPodsReady(ctx, client, "ns", "app=cluster-manager", 30, phase)
			if c.wantErr {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("waitUntilPodsReady() error = %v, want deadline exceeded", err)
				}
			} else if err != nil {
				t.Fatalf("waitUntilPodsReady() error = %v, want nil", err)
			}
			if got := phase.Load().(string); got != c.wantPhase {
				t.Fatalf("phase = %q, want %q", got, c.wantPhase)
			}
		})
	}
}

func newStatus(initial string) *atomic.Value {
	status := &atomic.Value{}
	status.Store(initial)
	return status
}

func testPod(name string, phase corev1.PodPhase, ready corev1.ConditionStatus, waiting string, deleting bool) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "ns",
			Labels:    map[string]string{"app": "cluster-manager"},
		},
		Status: corev1.PodStatus{
			Phase: phase,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: ready},
			},
		},
	}
	if waiting != "" {
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
			State: corev1.ContainerState{
				Waiting: &corev1.ContainerStateWaiting{Reason: waiting},
			},
		}}
	}
	if deleting {
		now := metav1.Now()
		pod.DeletionTimestamp = &now
		pod.Finalizers = []string{"test.finalizer"}
	}
	return pod
}

func TestTimeoutError(t *testing.T) {
	t.Parallel()

	if err := TimeoutError(nil, nil, "timed out waiting"); err != nil {
		t.Fatalf("TimeoutError(nil) = %v, want nil", err)
	}

	other := errors.New("boom")
	if err := TimeoutError(other, nil, "timed out waiting"); err != other {
		t.Fatalf("TimeoutError(other) = %v, want original error", err)
	}

	err := TimeoutError(context.DeadlineExceeded, nil, "timed out waiting")
	if err == nil || err.Error() != "timed out waiting: context deadline exceeded" {
		t.Fatalf("TimeoutError(deadline, empty) = %v, want timed out waiting", err)
	}

	status := &atomic.Value{}
	status.Store("connection refused")
	err = TimeoutError(context.DeadlineExceeded, status, "timed out waiting for %s", "operator")
	want := "timed out waiting for operator (connection refused): context deadline exceeded"
	if err == nil || err.Error() != want {
		t.Fatalf("TimeoutError() = %v, want %s", err, want)
	}
}
