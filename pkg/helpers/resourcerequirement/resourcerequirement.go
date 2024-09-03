// Copyright Contributors to the Open Cluster Management project

package resourcerequirement

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	operatorv1 "open-cluster-management.io/api/operator/v1"
)

func NewResourceRequirement(resourceType operatorv1.ResourceQosClass, limits, requests map[string]string) (*operatorv1.ResourceRequirement, error) {
	if len(limits)+len(requests) == 0 {
		if resourceType == operatorv1.ResourceQosClassResourceRequirement {
			return nil, fmt.Errorf("resource type is %s but both limits and requests are not set", resourceType)
		}
		return &operatorv1.ResourceRequirement{
			Type: resourceType,
		}, nil
	}
	if resourceType == "" {
		resourceType = operatorv1.ResourceQosClassResourceRequirement
	} else if resourceType != operatorv1.ResourceQosClassResourceRequirement {
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
	return &operatorv1.ResourceRequirement{
		Type:                 resourceType,
		ResourceRequirements: rr,
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
