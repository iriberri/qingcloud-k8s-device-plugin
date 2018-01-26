# Kubernets QingCloud Device Plugin

This is a [kubernets device plugin](https://kubernetes.io/docs/concepts/cluster-administration/device-plugins/) which aims to help qingcloud user to utilize GPU functionality with ease.

Following resources are implemented. if you need more resource please raise issue and let us know

|name |ResourceName|
GPU |nvidia.com/gpu

e.g.:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: demo-pod
spec:
  containers:
    -
      name: demo-container-1
      image: dockerhub.qingcloud.com/google_containers/pause:2.0
      resources:
        limits:
          nvidia.com/gpu: 2 # requesting 2 GPU
```

## Installation

The plugin will listen to the unix socket under /var/lib/kubelet/device-plugins/ by default and you can change it to any path you want

