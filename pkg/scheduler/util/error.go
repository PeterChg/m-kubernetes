package util

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
)

// InsufficientResource describes what kind of resource limit is hit and caused the pod to not fit the node.
type InsufficientResource struct {
	ResourceName v1.ResourceName
	// We explicitly have a parameter for reason to avoid formatting a message on the fly
	// for common resources, which is expensive for cluster autoscaler simulations.
	Reason    string
	Requested int64
	Used      int64
	Capacity  int64
	GpuType   string
}

func NewInsufficientResourceError(resourceName v1.ResourceName, reason string, requested, used, capacity int64, gpuType string) *InsufficientResource {
	return &InsufficientResource{
		ResourceName: resourceName,
		Reason:       reason,
		Requested:    requested,
		Used:         used,
		Capacity:     capacity,
		GpuType:      gpuType,
	}
}

func (e *InsufficientResource) Error() string {
	return fmt.Sprintf("Node didn't have enough resource: %s, requested: %d, used: %d, capacity: %d",
		e.ResourceName, e.Requested, e.Used, e.Capacity)
}

// GetReason returns the reason of the InsufficientResourceError.
func (e *InsufficientResource) GetReason() string {
	return fmt.Sprintf("Insufficient %v", e.ResourceName)
}

// GetInsufficientAmount returns the amount of the insufficient resource of the error.
func (e *InsufficientResource) GetInsufficientAmount() int64 {
	return e.Requested - (e.Capacity - e.Used)
}

// GetFreeAmount returns the unoccupied of the insufficient resource
func (e *InsufficientResource) GetFreeAmount() int64 {
	return e.Capacity - e.Used
}

func (e *InsufficientResource) SetGpuType(gpuType string) *InsufficientResource {
	if gpuType != "" {
		e.GpuType = gpuType
	}
	return e
}
