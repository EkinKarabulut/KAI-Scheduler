// Copyright 2025 NVIDIA CORPORATION
// SPDX-License-Identifier: Apache-2.0

package trainjob

import (
	"strconv"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/NVIDIA/KAI-scheduler/pkg/podgrouper/podgroup"
	"github.com/NVIDIA/KAI-scheduler/pkg/podgrouper/podgrouper/plugins/defaultgrouper"
)

const (
	// MinAvailableAnnotationKey is the annotation key where Kubeflow Trainer
	// passes the pre-calculated MinAvailable value.
	MinAvailableAnnotationKey = "kai.scheduler/min-available"
)

type TrainJobGrouper struct {
	*defaultgrouper.DefaultGrouper
}

func NewTrainJobGrouper(defaultGrouper *defaultgrouper.DefaultGrouper) *TrainJobGrouper {
	return &TrainJobGrouper{
		DefaultGrouper: defaultGrouper,
	}
}

func (tg *TrainJobGrouper) Name() string {
	return "Kubeflow TrainJob Grouper"
}

// +kubebuilder:rbac:groups=trainer.kubeflow.org,resources=trainjobs,verbs=get;list;watch
// +kubebuilder:rbac:groups=trainer.kubeflow.org,resources=trainjobs/finalizers,verbs=patch;update;create

func (tg *TrainJobGrouper) GetPodGroupMetadata(
	topOwner *unstructured.Unstructured, pod *v1.Pod, _ ...*metav1.PartialObjectMetadata,
) (*podgroup.Metadata, error) {
	podGroupMetadata, err := tg.DefaultGrouper.GetPodGroupMetadata(topOwner, pod)
	if err != nil {
		return nil, err
	}

	if minAvailStr, ok := pod.Annotations[MinAvailableAnnotationKey]; ok && minAvailStr != "" {
		minAvail, err := strconv.ParseInt(minAvailStr, 10, 32)
		if err == nil && minAvail > 0 {
			podGroupMetadata.MinAvailable = int32(minAvail)
		}
	}

	return podGroupMetadata, nil
}