Network of Mesos Containerizer
----

Mesos Containerizer use `network/cni` isolator to do network isolation.
`network/cni` implemented [CNI](09_network_cni.md).

Frameworks can specify the CNI network to which they want their containers to be attached by setting the name `name`
field in the `NetworkInfo` protobuf. This field is added into `NetworkInfo` from version 1.0.0, current of `mesos-go`
do not support this field yet. So we can not write a demo.