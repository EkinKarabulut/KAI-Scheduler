// Copyright 2025 NVIDIA CORPORATION
// SPDX-License-Identifier: Apache-2.0

package kubeflow

import (
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/util/slice"

	"github.com/NVIDIA/KAI-scheduler/pkg/scheduler/api/pod_info"
	"github.com/NVIDIA/KAI-scheduler/pkg/scheduler/framework"
)

const (
	podRoleLabelKey = "training.kubeflow.org/job-role"
	// jobCompletionIndexAnnotation is the annotation added by Kubernetes to pods
	// in indexed Jobs. For Kubeflow Trainer v2 TrainJob (which uses JobSet),
	// index 0 is the coordinator node.
	jobCompletionIndexAnnotation = "batch.kubernetes.io/job-completion-index"
)

var (
	masterRoleValues = []string{"master", "launcher"}
)

type kubeflowPlugin struct{}

func New(_ framework.PluginArguments) framework.Plugin {
	return &kubeflowPlugin{}
}

func (pp *kubeflowPlugin) Name() string {
	return "kubeflow"
}

func (pp *kubeflowPlugin) OnSessionOpen(ssn *framework.Session) {
	ssn.AddTaskOrderFn(TaskOrderFn)
}

func TaskOrderFn(l, r interface{}) int {
	lv := l.(*pod_info.PodInfo)
	rv := r.(*pod_info.PodInfo)

	lPodRole, lLabelExists := lv.Pod.Labels[podRoleLabelKey]
	rPodRole, rLabelExists := rv.Pod.Labels[podRoleLabelKey]

	lPodMasterRole := lLabelExists && slice.ContainsString(masterRoleValues, lPodRole, nil)
	rPodMasterRole := rLabelExists && slice.ContainsString(masterRoleValues, rPodRole, nil)

	if lPodMasterRole && !rPodMasterRole {
		return -1
	}
	if !lPodMasterRole && rPodMasterRole {
		return 1
	}

	if lLabelExists || rLabelExists {
		return 0
	}

	// Index 0 = coordinator (should be preempted last)
	lIndex := getJobCompletionIndex(lv.Pod)
	rIndex := getJobCompletionIndex(rv.Pod)

	if lIndex >= 0 || rIndex >= 0 {
		lIsCoordinator := lIndex == 0
		rIsCoordinator := rIndex == 0

		if lIsCoordinator && !rIsCoordinator {
			return -1
		}
		if !lIsCoordinator && rIsCoordinator {
			return 1
		}
	}

	return 0
}

func getJobCompletionIndex(pod *v1.Pod) int {
	if pod == nil || pod.Annotations == nil {
		return -1
	}

	indexStr, exists := pod.Annotations[jobCompletionIndexAnnotation]
	if !exists {
		return -1
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return -1
	}

	return index
}

func (pp *kubeflowPlugin) OnSessionClose(_ *framework.Session) {}
