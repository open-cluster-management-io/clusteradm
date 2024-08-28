// Copyright Contributors to the Open Cluster Management project

package resourcerequirement

import (
	_ "embed"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	operatorv1 "open-cluster-management.io/api/operator/v1"
)

func TestResourceRequirement(t *testing.T) {
	expectedLimitsCpu := "200m"
	expectedRequestsCpu := "100m"
	expectedLimitsMemory := "256Mi"
	expectedRequestsMemory := "128Mi"

	limits := map[string]string{
		"cpu":    expectedLimitsCpu,
		"memory": expectedLimitsMemory,
	}
	requests := map[string]string{
		"cpu":    expectedRequestsCpu,
		"memory": expectedRequestsMemory,
	}
	r, err := NewResourceRequirement(operatorv1.ResourceQosClassResourceRequirement, limits, requests)
	if err != nil {
		t.Fatalf("failed to create resource requirement: %v", err)
	}
	if r.ResourceRequirements.Limits.Cpu().String() != expectedLimitsCpu {
		t.Fatalf("expect limits.cpu is %s, but got %s", expectedLimitsCpu, r.ResourceRequirements.Limits.Cpu().String())
	}
	if r.ResourceRequirements.Requests.Cpu().String() != expectedRequestsCpu {
		t.Fatalf("expect requests.cpu is %s, but got %s", expectedRequestsCpu, r.ResourceRequirements.Requests.Cpu().String())
	}
	if r.ResourceRequirements.Limits.Memory().String() != expectedLimitsMemory {
		t.Fatalf("expect limits.memory to be %s, but got %s", expectedLimitsMemory, r.ResourceRequirements.Limits.Memory().String())
	}
	if r.ResourceRequirements.Requests.Memory().String() != expectedRequestsMemory {
		t.Fatalf("expect requests.memory to be %s, but got %s", expectedRequestsMemory, r.ResourceRequirements.Requests.Memory().String())
	}
}

func TestEnsureQuantity(t *testing.T) {
	tests := []struct {
		name     string
		limits   map[string]string
		requests map[string]string
		wantErr  bool
	}{
		{
			name: "requests less than limits",
			limits: map[string]string{
				"cpu":    "200m",
				"memory": "256Mi",
			},
			requests: map[string]string{
				"cpu":    "100m",
				"memory": "128Mi",
			},
			wantErr: false,
		},
		{
			name: "requests equal to limits",
			limits: map[string]string{
				"cpu":    "200m",
				"memory": "256Mi",
			},
			requests: map[string]string{
				"cpu":    "200m",
				"memory": "256Mi",
			},
			wantErr: false,
		},
		{
			name: "requests greater than limits",
			limits: map[string]string{
				"cpu":    "100m",
				"memory": "128Mi",
			},
			requests: map[string]string{
				"cpu":    "200m",
				"memory": "256Mi",
			},
			wantErr: true,
		},
		{
			name: "only limits but no requests",
			limits: map[string]string{
				"cpu":    "100m",
				"memory": "128Mi",
			},
			wantErr: false,
		},
		{
			name: "only requests but no limits",
			requests: map[string]string{
				"cpu":    "200m",
				"memory": "256Mi",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := &corev1.ResourceRequirements{
				Limits:   corev1.ResourceList{},
				Requests: corev1.ResourceList{},
			}
			for rsc, quantityStr := range tt.limits {
				quantity, _ := resource.ParseQuantity(quantityStr)
				rr.Limits[corev1.ResourceName(rsc)] = quantity
			}
			for rsc, quantityStr := range tt.requests {
				quantity, _ := resource.ParseQuantity(quantityStr)
				rr.Requests[corev1.ResourceName(rsc)] = quantity
			}
			if err := ensureQuantity(rr); (err != nil) != tt.wantErr {
				t.Errorf("ensureQuantity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
