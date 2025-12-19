// Copyright 2025 NVIDIA CORPORATION
// SPDX-License-Identifier: Apache-2.0

package trainjob

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/NVIDIA/KAI-scheduler/pkg/podgrouper/podgroup"
	"github.com/NVIDIA/KAI-scheduler/pkg/podgrouper/podgrouper/plugins/defaultgrouper"
)

type TrainJobGrouper struct {
	client client.Client
	*defaultgrouper.DefaultGrouper
}

func NewTrainJobGrouper(client client.Client, defaultGrouper *defaultgrouper.DefaultGrouper) *TrainJobGrouper {
	return &TrainJobGrouper{
		client:         client,
		DefaultGrouper: defaultGrouper,
	}
}

func (tg *TrainJobGrouper) Name() string {
	return "Kubeflow TrainJob Grouper"
}

// +kubebuilder:rbac:groups=trainer.kubeflow.org,resources=trainjobs,verbs=get;list;watch
// +kubebuilder:rbac:groups=trainer.kubeflow.org,resources=trainjobs/finalizers,verbs=patch;update;create
// +kubebuilder:rbac:groups=jobset.x-k8s.io,resources=jobsets,verbs=get;list;watch

func (tg *TrainJobGrouper) GetPodGroupMetadata(
	topOwner *unstructured.Unstructured, pod *v1.Pod, _ ...*metav1.PartialObjectMetadata,
) (*podgroup.Metadata, error) {
	podGroupMetadata, err := tg.DefaultGrouper.GetPodGroupMetadata(topOwner, pod)
	if err != nil {
		return nil, err
	}
	
	minAvailable, err := tg.calculateMinAvailableFromJobSet(topOwner.GetNamespace(), topOwner.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to calculate MinAvailable from JobSet: %w", err)
	}

	if minAvailable > 0 {
		podGroupMetadata.MinAvailable = minAvailable
	}

	return podGroupMetadata, nil
}

func (tg *TrainJobGrouper) calculateMinAvailableFromJobSet(namespace, name string) (int32, error) {
	jobSet := &unstructured.Unstructured{}
	jobSet.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "jobset.x-k8s.io",
		Kind:    "JobSet",
		Version: "v1alpha2",
	})

	err := tg.client.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, jobSet)
	if err != nil {
		return 0, fmt.Errorf("failed to get JobSet %s/%s: %w", namespace, name, err)
	}

	replicatedJobs, found, err := unstructured.NestedSlice(jobSet.Object, "spec", "replicatedJobs")
	if err != nil {
		return 0, fmt.Errorf("failed to get spec.replicatedJobs from JobSet %s/%s: %w", namespace, name, err)
	}
	if !found {
		return 0, fmt.Errorf("spec.replicatedJobs not found in JobSet %s/%s", namespace, name)
	}

	var totalPods int32
	for i, rj := range replicatedJobs {
		replicatedJob, ok := rj.(map[string]interface{})
		if !ok {
			return 0, fmt.Errorf("invalid structure of spec.replicatedJobs[%d] in JobSet %s/%s", i, namespace, name)
		}

		replicas, found, err := unstructured.NestedInt64(replicatedJob, "replicas")
		if err != nil {
			return 0, fmt.Errorf("failed to get replicas from spec.replicatedJobs[%d] in JobSet %s/%s: %w", i, namespace, name, err)
		}
		if !found {
			replicas = 1
		}

		parallelism, found, err := unstructured.NestedInt64(replicatedJob, "template", "spec", "parallelism")
		if err != nil {
			return 0, fmt.Errorf("failed to get parallelism from spec.replicatedJobs[%d] in JobSet %s/%s: %w", i, namespace, name, err)
		}
		if !found {
			parallelism = 1
		}

		totalPods += int32(replicas * parallelism)
	}

	return totalPods, nil
}
