# dummy-controller
Dummy controller manage the Dummy Custom Resource.

## Description
Dummy controller copy message from `dummySpec.Message` field to `dummyStatus.SpecEcho` field.
Also they create a pod, running nginx, and tracks the pod status in field `dummyStatus.PodStatus`. If we delete the CR - nginx pod will be deleted too. To specify the version of nginx please use env `POD_IMAGE`.


## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster using operator-sdk toolkit
1. Deploy the controller to the cluster:

```sh
make deploy
```

2. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/dummy_v1alpha1_dummy.yaml
```