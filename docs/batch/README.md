# Batch Scheduling
KAI Scheduler supports scheduling different types of workloads. Some workloads are scheduled as individual pods, while others require gang scheduling, meaning either all pods are scheduled together or none are scheduled until resources become available.

## BatchJob
To run a simple batch job with multiple pods that will be scheduled separately, run the following command:
```
kubectl apply -f batch-job.yaml
```
This will create 2 pods that will be scheduled separately. Both pods will either run at the same time or sequentially, depending on the available resources in the cluster.


## PyTorchJob (Kubeflow Training Operator v1)
To run in a distributed way across multiple pods, you can use PyTorchJob.

### Prerequisites
This requires the [kubeflow-training-operator-v1](https://www.kubeflow.org/docs/components/trainer/legacy-v1/) to be installed in the cluster.

### Instructions
Apply the following command to create a sample PyTorchJob with a master pod and two worker pods:
```
kubectl apply -f pytorch-job.yaml
```
Since gang scheduling is used, all 3 pods will be scheduled together, or none will be scheduled until resources become available in the cluster. 

## TrainJob (Kubeflow Trainer v2)
TrainJob is the new API from Kubeflow Trainer v2 for distributed training. It uses JobSets under the hood and provides a simpler, more unified experience across ML frameworks.

### Prerequisites
This requires [Kubeflow Trainer v2](https://www.kubeflow.org/docs/components/trainer/) to be installed in the cluster.

### Instructions
Apply the following command to create a distributed training job with 2 nodes:
```
kubectl apply -f train-job.yaml
```
This creates:
- A `ClusterTrainingRuntime` that defines the training configuration with KAI Scheduler
- A `TrainJob` that uses the runtime and specifies 2 nodes

Since gang scheduling is used, both pods will be scheduled together, or none will be scheduled until resources become available in the cluster.

