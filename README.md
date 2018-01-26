# Kubernets QingCloud Device Plugin

This is a [kubernets device plugin](https://kubernetes.io/docs/concepts/cluster-administration/device-plugins/) which aims to help qingcloud user utilize GPU functionality with ease.

Following resources are implemented. if you need more resource please raise issue and let us know

|name 			| ResourceName   |
| ------------- | -------------  |
|GPU 			| nvidia.com/gpu |

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

The plugin is a daemon process and will listen to the unix socket under /var/lib/kubelet/device-plugins/

to launch the process you can simply run commands "qingcloud-gpu", other options are listed as below

```bash
./qingcloud-gpu -h
Device plugin to help kubernetes cluster utilize GPU resources on Qingcloud

Usage:
  qingcloud-gpu [flags]

Flags:
      --alsologtostderr                  log to standard error as well as files
      --container-path string            Path on the container that mounts '-host-path' (default "/usr/local/nvidia")
  -h, --help                             help for qingcloud-gpu
      --host-path string                 Path on the host that contains nvidia libraries. This will be mounted inside the container as '-container-path' (default "/home/kubernetes/bin/nvidia")
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging

```

Happy coding!
