// Copyright Contributors to the Open Cluster Management project

package resourcerequirement

import (
	_ "embed"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/openshift/library-go/pkg/assets"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	operatorv1 "open-cluster-management.io/api/operator/v1"
)

//go:embed fake_deployment.yaml
var fakeDeploymentYaml string

type values struct {
	ResourceRequirement ResourceRequirement
}

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
	r, err := NewResourceRequirement(string(operatorv1.ResourceQosClassResourceRequirement), limits, requests)
	if err != nil {
		t.Fatalf("failed to create resource requirement: %v", err)
	}
	val := values{
		ResourceRequirement: *r,
	}
	data := assets.MustCreateAssetFromTemplate("fake-deployment", []byte(fakeDeploymentYaml), val).Data
	deployment := &v1.Deployment{}
	if err := yaml.Unmarshal(data, deployment); err != nil {
		t.Fatalf("failed to unmarshal deployment: %v", err)
	}
	resources := deployment.Spec.Template.Spec.Containers[0].Resources
	if resources.Limits.Cpu().String() != expectedLimitsCpu {
		t.Fatalf("expect limits.cpu is %s, but got %s", expectedLimitsCpu, resources.Limits.Cpu().String())
	}
	if resources.Requests.Cpu().String() != expectedRequestsCpu {
		t.Fatalf("expect requests.cpu is %s, but got %s", expectedRequestsCpu, resources.Requests.Cpu().String())
	}
	if resources.Limits.Memory().String() != expectedLimitsMemory {
		t.Fatalf("expect limits.memory to be %s, but got %s", expectedLimitsMemory, resources.Limits.Memory().String())
	}
	if resources.Requests.Memory().String() != expectedRequestsMemory {
		t.Fatalf("expect requests.memory to be %s, but got %s", expectedRequestsMemory, resources.Requests.Memory().String())
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
