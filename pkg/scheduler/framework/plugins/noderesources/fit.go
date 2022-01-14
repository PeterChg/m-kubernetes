/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package noderesources

import (
	"context"
	"fmt"
	v1helper "k8s.io/kubernetes/pkg/apis/core/v1/helper"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/util"
)

var _ framework.PreFilterPlugin = &Fit{}
var _ framework.FilterPlugin = &Fit{}
var _ framework.EnqueueExtensions = &Fit{}

const (
	// FitName is the name of the plugin used in the plugin registry and configurations.
	FitName = "NodeResourcesFit"

	// preFilterStateKey is the key in CycleState to NodeResourcesFit pre-computed data.
	// Using the name of the plugin will likely help us avoid collisions with other plugins.
	preFilterStateKey = "PreFilter" + FitName
)

// Fit is a plugin that checks if a node has sufficient resources.
type Fit struct {
	ignoredResources      sets.String
	ignoredResourceGroups sets.String
}

// preFilterState computed at PreFilter and used at Filter.
type preFilterState struct {
	framework.Resource
}

// Clone the prefilter state.
func (s *preFilterState) Clone() framework.StateData {
	return s
}

// Name returns name of the plugin. It is used in logs, etc.
func (f *Fit) Name() string {
	return FitName
}

func validateFitArgs(args config.NodeResourcesFitArgs) error {
	var allErrs field.ErrorList
	resPath := field.NewPath("ignoredResources")
	for i, res := range args.IgnoredResources {
		path := resPath.Index(i)
		if errs := metav1validation.ValidateLabelName(res, path); len(errs) != 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	groupPath := field.NewPath("ignoredResourceGroups")
	for i, group := range args.IgnoredResourceGroups {
		path := groupPath.Index(i)
		if strings.Contains(group, "/") {
			allErrs = append(allErrs, field.Invalid(path, group, "resource group name can't contain '/'"))
		}
		if errs := metav1validation.ValidateLabelName(group, path); len(errs) != 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if len(allErrs) == 0 {
		return nil
	}
	return allErrs.ToAggregate()
}

// NewFit initializes a new plugin and returns it.
func NewFit(plArgs runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	args, err := getFitArgs(plArgs)
	if err != nil {
		return nil, err
	}

	if err := validateFitArgs(args); err != nil {
		return nil, err
	}

	return &Fit{
		ignoredResources:      sets.NewString(args.IgnoredResources...),
		ignoredResourceGroups: sets.NewString(args.IgnoredResourceGroups...),
	}, nil
}

func getFitArgs(obj runtime.Object) (config.NodeResourcesFitArgs, error) {
	ptr, ok := obj.(*config.NodeResourcesFitArgs)
	if !ok {
		return config.NodeResourcesFitArgs{}, fmt.Errorf("want args to be of type NodeResourcesFitArgs, got %T", obj)
	}
	return *ptr, nil
}

// computePodResourceRequest returns a framework.Resource that covers the largest
// width in each resource dimension. Because init-containers run sequentially, we collect
// the max in each dimension iteratively. In contrast, we sum the resource vectors for
// regular containers since they run simultaneously.
//
// If Pod Overhead is specified and the feature gate is set, the resources defined for Overhead
// are added to the calculated Resource request sum
//
// Example:
//
// Pod:
//   InitContainers
//     IC1:
//       CPU: 2
//       Memory: 1G
//     IC2:
//       CPU: 2
//       Memory: 3G
//   Containers
//     C1:
//       CPU: 2
//       Memory: 1G
//     C2:
//       CPU: 1
//       Memory: 1G
//
// Result: CPU: 3, Memory: 3G
func computePodResourceRequest(pod *v1.Pod) *preFilterState {
	result := &preFilterState{}
	for _, container := range pod.Spec.Containers {
		result.Add(container.Resources.Requests)
	}

	// take max_resource(sum_pod, any_init_container)
	for _, container := range pod.Spec.InitContainers {
		result.SetMaxResource(container.Resources.Requests)
	}

	// If Overhead is being utilized, add to the total requests for the pod
	if pod.Spec.Overhead != nil && utilfeature.DefaultFeatureGate.Enabled(features.PodOverhead) {
		result.Add(pod.Spec.Overhead)
	}

	return result
}

// PreFilter invoked at the prefilter extension point.
func (f *Fit) PreFilter(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod) *framework.Status {
	cycleState.Write(preFilterStateKey, computePodResourceRequest(pod))
	return nil
}

// PreFilterExtensions returns prefilter extensions, pod add and remove.
func (f *Fit) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func getPreFilterState(cycleState *framework.CycleState) (*preFilterState, error) {
	c, err := cycleState.Read(preFilterStateKey)
	if err != nil {
		// preFilterState doesn't exist, likely PreFilter wasn't invoked.
		return nil, fmt.Errorf("error reading %q from cycleState: %w", preFilterStateKey, err)
	}

	s, ok := c.(*preFilterState)
	if !ok {
		return nil, fmt.Errorf("%+v  convert to NodeResourcesFit.preFilterState error", c)
	}
	return s, nil
}

// EventsToRegister returns the possible events that may make a Pod
// failed by this plugin schedulable.
// NOTE: if in-place-update (KEP 1287) gets implemented, then PodUpdate event
// should be registered for this plugin since a Pod update may free up resources
// that make other Pods schedulable.
func (f *Fit) EventsToRegister() []framework.ClusterEvent {
	return []framework.ClusterEvent{
		{Resource: framework.Pod, ActionType: framework.Delete},
		{Resource: framework.Node, ActionType: framework.Add | framework.UpdateNodeAllocatable},
	}
}

// Filter invoked at the filter extension point.
// Checks if a node has sufficient resources, such as cpu, memory, gpu, opaque int resources etc to run a pod.
// It returns a list of insufficient resources, if empty, then the node has all the resources requested by the pod.
func (f *Fit) Filter(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	s, err := getPreFilterState(cycleState)
	if err != nil {
		return framework.AsStatus(err)
	}

	insufficientResources := fitsRequest(s, nodeInfo, f.ignoredResources, f.ignoredResourceGroups)

	if len(insufficientResources) != 0 {
		// We will keep all failure reasons.
		failureReasons := make([]string, 0, len(insufficientResources))
		for _, r := range insufficientResources {
			failureReasons = append(failureReasons, r.Reason)
		}
		status := framework.NewStatus(framework.Unschedulable, failureReasons...)
		status.SetResourceInsufficientInfo(insufficientResources)
		return status
	}
	return nil
}

//// InsufficientResource describes what kind of resource limit is hit and caused the pod to not fit the node.
//type InsufficientResource struct {
//	ResourceName v1.ResourceName
//	// We explicitly have a parameter for reason to avoid formatting a message on the fly
//	// for common resources, which is expensive for cluster autoscaler simulations.
//	Reason    string
//	Requested int64
//	Used      int64
//	Capacity  int64
//	GpuType   string
//}
//
//func NewInsufficientResourceError(resourceName v1.ResourceName, reason string, requested, used, capacity int64, gpuType string) *InsufficientResource {
//	return &InsufficientResource{
//		ResourceName: resourceName,
//		Reason:       reason,
//		Requested:    requested,
//		Used:         used,
//		Capacity:     capacity,
//		GpuType:      gpuType,
//	}
//}
//
//func (e *InsufficientResource) Error() string {
//	return fmt.Sprintf("Node didn't have enough resource: %s, requested: %d, used: %d, capacity: %d",
//		e.ResourceName, e.Requested, e.Used, e.Capacity)
//}
//
//// GetReason returns the reason of the InsufficientResourceError.
//func (e *InsufficientResource) GetReason() string {
//	return fmt.Sprintf("Insufficient %v", e.ResourceName)
//}
//
//// GetInsufficientAmount returns the amount of the insufficient resource of the error.
//func (e *InsufficientResource) GetInsufficientAmount() int64 {
//	return e.Requested - (e.Capacity - e.Used)
//}
//
//// GetFreeAmount returns the unoccupied of the insufficient resource
//func (e *InsufficientResource) GetFreeAmount() int64 {
//	return e.Capacity - e.Used
//}
//
//func (e *InsufficientResource) SetGpuType(gpuType string) *InsufficientResource {
//	if gpuType != "" {
//		e.GpuType = gpuType
//	}
//	return e
//}

// Fits checks if node have enough resources to host the pod.
func Fits(pod *v1.Pod, nodeInfo *framework.NodeInfo) []*util.InsufficientResource {
	return fitsRequest(computePodResourceRequest(pod), nodeInfo, nil, nil)
}

func fitsRequest(podRequest *preFilterState, nodeInfo *framework.NodeInfo, ignoredExtendedResources, ignoredResourceGroups sets.String) []*util.InsufficientResource {
	insufficientResources := make([]*util.InsufficientResource, 0, 4)
	var gpuType string

	allowedPodNumber := nodeInfo.Allocatable.AllowedPodNumber
	if len(nodeInfo.Pods)+1 > allowedPodNumber {
		insufficientResources = append(insufficientResources, util.NewInsufficientResourceError(
			v1.ResourcePods,
			"Too many pods",
			1,
			int64(len(nodeInfo.Pods)),
			int64(allowedPodNumber),
			gpuType,
		))
	}

	if podRequest.MilliCPU == 0 &&
		podRequest.Memory == 0 &&
		podRequest.EphemeralStorage == 0 &&
		len(podRequest.ScalarResources) == 0 {
		return insufficientResources
	}

	for rName, rQuant := range podRequest.ScalarResources {
		if v1helper.IsExtendedResourceName(rName) {
			// If this resource is one of the extended resources that should be ignored, we will skip checking it.
			// rName is guaranteed to have a slash due to API validation.
			var rNamePrefix string
			if ignoredResourceGroups.Len() > 0 {
				rNamePrefix = strings.Split(string(rName), "/")[0]
			}
			if ignoredExtendedResources.Has(string(rName)) || ignoredResourceGroups.Has(rNamePrefix) {
				continue
			}
		}

		if strings.HasPrefix(string(rName), "cloudml.vgpu/") || strings.HasPrefix(string(rName), "cloudml.gpu/") {
			gpuType = string(rName)
		}

		if rQuant > (nodeInfo.Allocatable.ScalarResources[rName] - nodeInfo.Requested.ScalarResources[rName]) {
			insufficientResources = append(insufficientResources, util.NewInsufficientResourceError(
				rName,
				fmt.Sprintf("Insufficient %v", rName),
				podRequest.ScalarResources[rName],
				nodeInfo.Requested.ScalarResources[rName],
				nodeInfo.Allocatable.ScalarResources[rName],
				gpuType,
			))
		}
	}

	if podRequest.MilliCPU > (nodeInfo.Allocatable.MilliCPU - nodeInfo.Requested.MilliCPU) {
		insufficientResources = append(insufficientResources, util.NewInsufficientResourceError(
			v1.ResourceCPU,
			"Insufficient cpu",
			podRequest.MilliCPU,
			nodeInfo.Requested.MilliCPU,
			nodeInfo.Allocatable.MilliCPU,
			gpuType,
		))
	}
	if podRequest.Memory > (nodeInfo.Allocatable.Memory - nodeInfo.Requested.Memory) {
		insufficientResources = append(insufficientResources, util.NewInsufficientResourceError(
			v1.ResourceMemory,
			"Insufficient memory",
			podRequest.Memory,
			nodeInfo.Requested.Memory,
			nodeInfo.Allocatable.Memory,
			gpuType,
		))
	}
	if podRequest.EphemeralStorage > (nodeInfo.Allocatable.EphemeralStorage - nodeInfo.Requested.EphemeralStorage) {
		insufficientResources = append(insufficientResources, util.NewInsufficientResourceError(
			v1.ResourceEphemeralStorage,
			"Insufficient ephemeral-storage",
			podRequest.EphemeralStorage,
			nodeInfo.Requested.EphemeralStorage,
			nodeInfo.Allocatable.EphemeralStorage,
			gpuType,
		))
	}

	return insufficientResources
}
