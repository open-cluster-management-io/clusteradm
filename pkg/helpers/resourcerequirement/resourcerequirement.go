// Copyright Contributors to the Open Cluster Management project

package resourcerequirement

import (
	"fmt"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	operatorv1 "open-cluster-management.io/api/operator/v1"
)

// ResourceRequirement is for templating resource requirement
type ResourceRequirement struct {
	Type                 string
	ResourceRequirements []byte
}

func NewResourceRequirement(resourceType string, limits, requests map[string]string) (*ResourceRequirement, error) {
	if len(limits)+len(requests) == 0 {
		if resourceType == string(operatorv1.ResourceQosClassResourceRequirement) {
			return nil, fmt.Errorf("resource type is %s but both limits and requests are not set", resourceType)
		}
		return &ResourceRequirement{
			Type: resourceType,
		}, nil
	}
	if resourceType == "" {
		resourceType = string(operatorv1.ResourceQosClassResourceRequirement)
	} else if resourceType != string(operatorv1.ResourceQosClassResourceRequirement) {
		return nil, fmt.Errorf("resource type must be %s when resource limits or requests are set", string(operatorv1.ResourceQosClassResourceRequirement))
	}
	rr := &corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{},
		Requests: corev1.ResourceList{},
	}
	for rsc, quantityStr := range limits {
		quantity, err := resource.ParseQuantity(quantityStr)
		if err != nil {
			return nil, err
		}
		rr.Limits[corev1.ResourceName(rsc)] = quantity
	}
	for rsc, quantityStr := range requests {
		quantity, err := resource.ParseQuantity(quantityStr)
		if err != nil {
			return nil, err
		}
		rr.Requests[corev1.ResourceName(rsc)] = quantity
	}
	if err := ensureQuantity(rr); err != nil {
		return nil, err
	}
	marshal, err := yaml.Marshal(rr)
	if err != nil {
		return nil, err
	}
	return &ResourceRequirement{
		Type:                 resourceType,
		ResourceRequirements: marshal,
	}, nil
}

func ensureQuantity(r *corev1.ResourceRequirements) error {
	for rsc, limitsQuantity := range r.Limits {
		requestsQuantity, ok := r.Requests[rsc]
		if !ok {
			continue
		}
		if requestsQuantity.Cmp(limitsQuantity) <= 0 {
			continue
		}
		return fmt.Errorf("requests %s must be less than or equal to limits %s",
			requestsQuantity.String(), limitsQuantity.String())
	}
	return nil
}
