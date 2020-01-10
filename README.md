# Macvtap device plugin for Kubernetes

This device plugin provides one or more configurable resources that allocate
a macvtap interface from a specific interface on the node and provides the
associated tap device to the pod.

Each resource can be configured by lower link, operating mode and capacity.
If further details need to be configured on the device or if the interface
itself needs to be available in the pod, combine this device plugin with the
[macvtap cni](https://github.com/maiqueb/macvtap-cni).

Main use case is to run a VM within the pod that uses the tap device as network
backend.

This repository was originally forked from the soon to be or already archived
kubevirt's [kubernetes-device-plugins](https://github.com/kubevirt/kubernetes-device-plugins)

## Usage

> **Note:** When using this device plugin ensure to open the feature gate
> using `kubelet`'s `--feature-gates=DevicePlugins=true`

Macvtap device plugin is configured through environment variable
`DP_MACVTAP_CONF`. The value is a json array and each element of the array is
a separate resource to be made available:

* `name` (string, required) the name of the resource
* `master` (string, required) the name of the lower link
* `mode` (string, optional, default=bridge) the operating mode
* `capacity` (uint, optional, default=100) the capacity of the resource

The configuration is provided through a
[config map](manifests/macvtapdp-config.yml):

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: macvtapdp-config
data:
  DP_MACVTAP_CONF: |
    [ {
        "name" : "eth0",
        "master" : "eth0",
        "mode": "bridge",
        "capacity" : 50
     } ]
```

```bash
$ kubectl apply -f https://raw.githubusercontent.com/jcaamano/macvtap-device-plugin/master/manifests/macvtapdp-config.yml
configmap "macvtapdp-config" created
```

Once this is done, the device plugin can be deployed using
[daemon set](manifests/macvtapdp-daemonset.yml):

```
$ kubectl apply -f https://raw.githubusercontent.com/jcaamano/macvtap-device-plugin/master/manifests/macvtapdp-daemonset.yml
daemonset "macvtap-device-plugin" created

$ kubectl get pods
NAME                                 READY     STATUS    RESTARTS   AGE
macvtap-device-plugin-745x4            1/1    Running           0    5m
```

If the daemonset is running the user can define a i
[pod](examples/macvtap-consumer.yml)requesting a macvtap tap device over host
interface `eth0`.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: macvtap-consumer
spec:
  containers:
  - name: busybox
    image: busybox
    command: ["/bin/sleep", "123"]
    resources:
      requests:
        macvtap.network.kubevirt.io/eth0: 1
      limits:
        macvtap.network.kubevirt.io/eth0: 1
```

## Develop
This device plugin uses Device Plugin Manager framework. See documentation on
https://godoc.org/github.com/kubevirt/device-plugin-manager/pkg/dpm

Build:

```
make build
```

Test:

```
make test
```

Deploy local Kubernetes cluster:

```
make cluster-up
```

Destroy local Kubernetes cluster:

```
make cluster-down
```

Build device plugin images and push to local Kubernetes cluster registry:

```
make cluster-sync
```

Access cluster kubectl:

```
./cluster/kubectl.sh ...
```

Access cluster node via ssh:

```
./cluster/cli.sh node01
```

Run e2e tests (on running cluster):

```
make functests
```
