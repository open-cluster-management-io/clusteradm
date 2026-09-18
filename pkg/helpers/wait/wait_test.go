// Copyright Contributors to the Open Cluster Management project
package wait

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestPodReadyEventHandler(t *testing.T) {
	cases := []struct {
		name      string
		object    *corev1.Pod
		wantReady bool
		wantPhase string
	}{
		{
			name: "pod without ready condition is not ready",
			object: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
				},
			},
			wantReady: false,
			wantPhase: "Pending",
		},
		{
			name: "pod with ready condition false is not ready",
			object: &corev1.Pod{
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
			object: &corev1.Pod{
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
			object: &corev1.Pod{
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
		})
	}
}
